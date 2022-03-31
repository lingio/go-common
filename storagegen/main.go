package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/lingio/go-common"
	zl "github.com/rs/zerolog/log"
)

func main() {
	if len(os.Args) < 2 {
		zl.Fatal().Msg("Usage: go run main.go <spec.json>")
	}

	specFilepath := os.Args[1]
	spec := common.ReadStorageSpec(specFilepath)
	dir := path.Dir(specFilepath)
	generateStorage(dir, spec)
}

func generateStorage(dir string, spec common.ServiceStorageSpec) {
	defaultObjectStoreConfig := common.ObjectStoreConfig{
		ContentType:        "application/json",
		ContentDisposition: "",
	}

	for _, b := range spec.Buckets {
		privateTypeName := strings.ToLower(b.TypeName[0:1]) + b.TypeName[1:]
		idName := "ID"
		if b.IdName != nil {
			idName = *b.IdName
		}
		config := defaultObjectStoreConfig
		if b.Config != nil {
			config = *b.Config
		}
		getAll := false // default to false
		if b.GetAll != nil {
			getAll = *b.GetAll
		}
		// Patch secondary index default values
		for i, idx := range b.SecondaryIndexes {
			switch idx.Type {
			case common.INDEX_TYPE_UNIQUE:
				break
			case common.INDEX_TYPE_SET:
				break
			default:
				zl.Fatal().Msg("unknown index 'type': " + idx.Type)
			}

			if idx.Key == "" && len(idx.Keys) == 0 {
				log.Fatalln(fmt.Errorf("%s secondaryIndex[%d]: missing 'key' or 'keys'", b.TypeName, i))
			} else if idx.Key != "" && len(idx.Keys) > 0 {
				log.Fatalln(fmt.Errorf("%s secondaryIndex[%d]: cannot use both 'key' and 'keys'", b.TypeName, i))
			} else if idx.Key != "" && len(idx.Keys) == 0 {
				idx.Keys = append(idx.Keys, common.IndexComponent{
					Key:      idx.Key,
					Param:    "", // default to same name as Key
					Optional: false,
				})
			}

			// Ensure key is exported.
			for _, field := range idx.Keys {
				if field.Key[0] >= 'a' && field.Key[0] <= 'z' {
					log.Fatalln(fmt.Errorf("%s secondaryIndex[%d]: key '%s' is not exported", b.TypeName, i, field.Key))
				}
				if field.Optional {
					idx.Optional = true
				}
			}

			// By convention, compound indexes have the primary discriminant in the last position. E.g. [Partner, Email]
			lastKey := idx.Keys[len(idx.Keys)-1].Key
			// Default value for name is key: e.g. Get<All?>ByEmail
			if idx.Name == "" {
				idx.Name = lastKey
			}

			// Default cache key
			idx.CacheKey = strings.ToLower(lastKey[0:1]) + lastKey[1:]
			b.SecondaryIndexes[i] = idx
		}

		bytes := generate("tmpl/"+b.Template, TmplParams{
			ServiceName:      spec.ServiceName,
			TypeName:         b.TypeName,
			PrivateTypeName:  privateTypeName,
			DbTypeName:       b.DbTypeName,
			BucketName:       b.BucketName,
			SecondaryIndexes: b.SecondaryIndexes,
			IdName:           idName,
			Version:          b.Version,
			Config:           config,
			GetAll:           getAll,
			FilenameFormat:   b.FilenameFormat,
		})

		// go codeconv uses _ in filenames
		filename := fmt.Sprintf("%s.gen.go", pascalCase2SnakeCase(b.TypeName))
		filepath := path.Join(dir, filename)
		err := ioutil.WriteFile(filepath, bytes, 0644)
		if err != nil {
			zl.Fatal().Str("err", err.Error()).Msg("failed to load minio template")
		}
		if err := postprocess(filepath); err != nil {
			zl.Warn().Str("err", err.Error()).Str("file", filename).Msg("failed to format file")
		}
	}

	bytes := generate("tmpl/common.tmpl", TmplParams{})
	err := ioutil.WriteFile(fmt.Sprintf("%s/common.gen.go", dir), bytes, 0644)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Msg("failed to load common template")
	}
}

func pascalCase2SnakeCase(str string) string {
	var b []rune

	b = append(b, rune(str[0])+('a'-'A'))
	for _, c := range str[1:] {
		if c >= 'A' && c <= 'Z' {
			b = append(b, '_')
			b = append(b, c+('a'-'A'))
		} else if c == '-' {
			b = append(b, '_')
		} else {
			b = append(b, c)
		}
	}
	return string(b)
}

type TmplParams struct {
	TypeName         string
	PrivateTypeName  string
	DbTypeName       string
	BucketName       string
	ServiceName      string
	IdName           string
	Version          string
	FilenameFormat   string
	SecondaryIndexes []common.SecondaryIndex
	Config           common.ObjectStoreConfig
	GetAll           bool
}

func generate(tmplFilename string, params interface{}) []byte {
	funcMap := template.FuncMap{
		"ToUpper":       strings.ToUpper,
		"ToLower":       strings.ToLower,
		"PrettyPrint":   prettyPrint,
		"CamelCase":     camelCaseKey,
		"Join":          joinString,
		"CompareFields": compareFields,
		"Materialize":   materialize,
		"CheckOptional": checkOptionalField,
		"IndexKeysOnly": filterIndexKeys,
	}

	main := path.Base(tmplFilename)
	tpl, err := template.New(main).Funcs(funcMap).ParseFiles(tmplFilename)
	if err != nil {
		zl.Fatal().Str("tmplFilename", tmplFilename).Str("err", err.Error()).Msg("failed to load template")
	} else if tpl == nil {
		zl.Fatal().Str("tmplFilename", tmplFilename).Msg("template is nil. failed to load message template")
	}

	var b bytes.Buffer
	if err2 := tpl.Execute(&b, params); err2 != nil {
		zl.Fatal().Str("tmplFilename", tmplFilename).Str("err", err2.Error()).Msg("failed to generate message")
	}
	return b.Bytes()
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func camelCaseKey(keys []common.IndexComponent) []string {
	var s []string
	for _, idx := range keys {
		var p string
		if idx.Param != "" {
			p = idx.Param
		} else {
			p = idx.Key
		}
		s = append(s, strings.ToLower(p[0:1])+p[1:])
	}
	return s
}

func accessField(i common.IndexComponent, on string) string {
	if i.Optional {
		return fmt.Sprintf("*%s.%s", on, i.Key)
	}
	return fmt.Sprintf("%s.%s", on, i.Key)
}

// obj []indexes => [obj.Field1, *obj.Field2, ...]
func materialize(on string, fields []common.IndexComponent) []string {
	var s []string
	for _, idx := range fields {
		if !idx.ExclFromIndex {
			s = append(s, accessField(idx, on))
		}
	}
	return s
}

func filterIndexKeys(keys []common.IndexComponent) []common.IndexComponent {
	var s []common.IndexComponent
	for _, key := range keys {
		if key.ExclFromIndex {
			continue
		}
		s = append(s, key)
	}
	return s
}

func checkOptionalField(on string, fields []common.IndexComponent) []string {
	var s []string
	for _, idx := range fields {
		if idx.Optional {
			s = append(s, fmt.Sprintf("%s.%s != nil", on, idx.Key))
		}
	}
	return s
}

// orig obj != key => orig.key != obj.key
func compareFields(a, b, comp string, keys []common.IndexComponent) []string {
	var s []string
	for _, idx := range keys {
		// Skip excluded optional indexes since they will always return true or false:
		//   - *a.B != *b.B --> always true unless a.B and b.B points to same B
		//   - *a.B == *b.B --> always false unless a.B and b.B points to same B
		if idx.ExclFromIndex && idx.Optional {
			continue
		}
		s = append(s, accessField(idx, a)+comp+accessField(idx, b))
	}
	return s
}

func joinString(b string, a []string) string {
	return strings.Join(a, b)
}

func postprocess(filepath string) error {
	var gofmt bool
	var imports bool
	// If go is installed the standard way from https://golang.org/doc/install
	// then we will detect at least Linux and MacOS. Special case for Ubuntu snap.
	for _, gobin := range []string{"go", "/snap/bin/go", "/usr/local/go/bin/go"} {
		exe, err := exec.LookPath(gobin)
		if err != nil {
			continue
		}

		cmd := exec.Command(exe, "fmt", filepath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("format '%s': %w", filepath, err)
		}
		gofmt = true
		break
	}

	for _, goimp := range []string{"goimports", "/snap/bin/goimports", "/usr/local/go/bin/goimports"} {
		exe, err := exec.LookPath(goimp)
		if err != nil {
			continue
		}

		cmd := exec.Command(exe, "-w", "-srcdir", filepath, filepath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("goimport '%s': %w", filepath, err)
		}
		imports = true
		break
	}

	if !gofmt {
		zl.Warn().Str("file", filepath).Msg("skipping go fmt")
	}
	if !imports {
		zl.Warn().Str("file", filepath).Msg("skipping goimports")
	}
	return nil
}

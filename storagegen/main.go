package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/lingio/go-common"
	zl "github.com/rs/zerolog/log"
)

type BucketSpec struct {
	TypeName         string
	SecondaryIndexes []SecondaryIndex
	DbTypeName       string
	BucketName       string
	Template         string
	Version          string
	IdName           *string
	GetAll           *bool
	Config           *common.ObjectStoreConfig
}

type SecondaryIndex struct {
	Key, Type, Name, CacheKey string
	Optional                  bool
}

type StorageSpec struct {
	ServiceName string
	Buckets     []BucketSpec
}

type Statement struct {
	Effect   string
	Action   []string
	Resource []string
}

type MinioPolicy struct {
	Version   string
	Statement []Statement
}

func main() {
	//typeName := "UserStartedClasses"
	//dbTypeName := "UserStartedClasses"
	//bucketName := "user-started-classes"
	//fileName   := "user_started_classes.gen.go"

	if len(os.Args) < 2 {
		zl.Fatal().Msg("Usage: go run main.go <spec.json>")
	}
	spec := readSpec(os.Args[1])
	dir := path.Dir(os.Args[1])

	defaultObjectStoreConfig := common.ObjectStoreConfig{
		Versioning:         false,
		ObjectLocking:      false,
		Lifecycle:          nil,
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
			case "unique":
				fallthrough
			case "set":
				break
			default:
				zl.Fatal().Msg("unknown index type: " + idx.Type)
			}
			// Ensure key is exported.
			if idx.Key[0] >= 'a' && idx.Key[0] <= 'z' {
				zl.Fatal().Msg("index key must not be private: " + idx.Key)
			}
			// Default value for name is key: e.g. Get<All?>ByEmail
			if idx.Name == "" {
				idx.Name = idx.Key
			}
			// Default cache key
			idx.CacheKey = strings.ToLower(idx.Key[0:1]) + idx.Key[1:]
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
		})

		// go codeconv uses _ in filenames
		filename := fmt.Sprintf("%s.gen.go", strings.Replace(b.BucketName, "-", "_", -1))
		filepath := path.Join(dir, filename)
		err := ioutil.WriteFile(filepath, bytes, 0644)
		if err != nil {
			zl.Fatal().Str("err", err.Error()).Msg("failed to load minio template")
		}
	}

	bytes := generate("tmpl/common.tmpl", TmplParams{})
	err := ioutil.WriteFile(fmt.Sprintf("%s/common.gen.go", dir), bytes, 0644)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Msg("failed to load common template")
	}

	mp := &MinioPolicy{
		Version:   "2012-10-17",
		Statement: make([]Statement, 1),
	}
	mp.Statement[0].Effect = "Allow"
	mp.Statement[0].Action = []string{"s3:GetObject", "s3:PutObject", "s3:ListBucket"}
	mp.Statement[0].Resource = make([]string, 0)

	for _, b := range spec.Buckets {
		mp.Statement[0].Resource = append(mp.Statement[0].Resource, fmt.Sprintf("arn:aws:s3:::%s", b.BucketName))
		mp.Statement[0].Resource = append(mp.Statement[0].Resource, fmt.Sprintf("arn:aws:s3:::%s/*", b.BucketName))
	}
	bytes2, err := json.MarshalIndent(mp, "", "  ")
	if err != nil {
		zl.Fatal().Msg("failed marshalling to json")
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s/minio_policy.json", dir), bytes2, 0644)
	if err != nil {
		zl.Fatal().Msg("failed to write minio policy to file")
	}
}

func readSpec(filename string) StorageSpec {

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Str("filename", filename).Msg("failed to load storage spec file")
	}

	spec := StorageSpec{}
	err = json.Unmarshal(file, &spec)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Str("filename", filename).Msg("failed to unmarshal storage spec file")
	}
	return spec
}

type TmplParams struct {
	TypeName         string
	PrivateTypeName  string
	DbTypeName       string
	BucketName       string
	ServiceName      string
	IdName           string
	Version          string
	SecondaryIndexes []SecondaryIndex
	Config           common.ObjectStoreConfig
	GetAll           bool
}

type TmplParams2 struct {
	Buckets []BucketSpec
}

func generate(tmplFilename string, params interface{}) []byte {
	funcMap := template.FuncMap{
		"ToUpper":     strings.ToUpper,
		"ToLower":     strings.ToLower,
		"PrettyPrint": prettyPrint,
	}
	tpltxt, err := os.ReadFile(tmplFilename)
	if err != nil {
		zl.Fatal().Str("tmplFilename", tmplFilename).Str("err", err.Error()).Msg("failed to read template")
	}

	// tpl, err := template.ParseFiles(tmplFilename)
	tpl, err := template.New(path.Base(tmplFilename)).Funcs(funcMap).Parse(string(tpltxt))
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

package gen

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"text/template"

	zl "github.com/rs/zerolog/log"
)

type Func struct {
	TmplParams TmplParams
	HttpMethod string
}

type TmplParams struct {
	Path         string
	PathTemplate string
	FuncName     string
	BodyType     string
	RetObjType   string
	Params       string
	Params2      string
	QueryParams  []QueryParam
	PackageName  string
	TokenAuth    bool
	ApiKeyAuth   bool
}

func Postfix(f Func) string {
	postfix := ""

	if f.TmplParams.Params == "" {
		postfix += "NoParams"
	} else if f.TmplParams.Params2 == "" {
		postfix += "NoParams2"
	}
	if (f.HttpMethod == "POST" || f.HttpMethod == "PUT" || f.HttpMethod == "DELETE" || f.HttpMethod == "PATCH") && f.TmplParams.BodyType == "" {
		postfix += "NoBody"
	}
	return postfix
}

func GenerateFromSpec(tfs fs.FS, es ExtSpec, specFilename string, outdir string) {
	b := make([]byte, 0)
	b = append(b, generateBeginning(tfs, es.Package)...)
	funcMap := ReadSpec(specFilename)
	for _, fs := range es.OpenOperations {
		f, ok := funcMap[fs]
		if !ok {
			zl.Fatal().Str("operationID", fs).Msg("operationID not found")
		}
		if !f.TmplParams.TokenAuth && !f.TmplParams.ApiKeyAuth {
			b = append(b, generate(tfs, fmt.Sprintf("tmpl/%s/%s%s.tmpl", "none", strings.ToLower(f.HttpMethod), Postfix(f)), f.TmplParams)...)
		} else {
			zl.Fatal().Bool("tokenAuth", f.TmplParams.TokenAuth).Bool("apiKeyAuth", f.TmplParams.ApiKeyAuth).Msg("auth mismatch, expected none")
		}
	}
	for _, fs := range es.TokenOperations {
		f, ok := funcMap[fs]
		if !ok {
			zl.Fatal().Str("operationID", fs).Msg("operationID not found")
		}
		if f.TmplParams.TokenAuth {
			b = append(b, generate(tfs, fmt.Sprintf("tmpl/%s/%s%s.tmpl", "token", strings.ToLower(f.HttpMethod), Postfix(f)), f.TmplParams)...)
		} else {
			zl.Fatal().Bool("tokenAuth", f.TmplParams.TokenAuth).Bool("apiKeyAuth", f.TmplParams.ApiKeyAuth).Msg("auth mismatch, expected token")
		}
	}
	for _, fs := range es.ApiKeyOperations {
		f, ok := funcMap[fs]
		if !ok {
			zl.Fatal().Str("operationID", fs).Msg("operationID not found")
		}
		if f.TmplParams.ApiKeyAuth {
			b = append(b, generate(tfs, fmt.Sprintf("tmpl/%s/%s%s.tmpl", "apikey", strings.ToLower(f.HttpMethod), Postfix(f)), f.TmplParams)...)
		} else {
			zl.Fatal().Bool("tokenAuth", f.TmplParams.TokenAuth).Bool("apiKeyAuth", f.TmplParams.ApiKeyAuth).Msg("auth mismatch, expected token")
		}
	}
	err := os.WriteFile(fmt.Sprintf("%s/%s", outdir, "client.gen.go"), b, 0644)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Msg("error writing file")
	}
}

func GenerateAll(tfs fs.FS, funcs []Func, outdir string, packageName string, clientFilename string) {
	b := make([]byte, 0)
	b = append(b, generateBeginning(tfs, packageName)...)

	for _, f := range funcs {
		postfix := ""
		if f.TmplParams.Params == "" {
			postfix += "NoParams"
		}
		if (f.HttpMethod == "POST" || f.HttpMethod == "PUT" || f.HttpMethod == "DELETE") && f.TmplParams.BodyType == "" {
			postfix += "NoBody"
		}
		if f.TmplParams.TokenAuth {
			b = append(b, generate(tfs, fmt.Sprintf("tmpl/%s/%s%s.tmpl", "token", strings.ToLower(f.HttpMethod), postfix), f.TmplParams)...)
		}
		if f.TmplParams.ApiKeyAuth {
			b = append(b, generate(tfs, fmt.Sprintf("tmpl/%s/%s%s.tmpl", "apikey", strings.ToLower(f.HttpMethod), postfix), f.TmplParams)...)
		}
		if !f.TmplParams.TokenAuth && !f.TmplParams.ApiKeyAuth {
			b = append(b, generate(tfs, fmt.Sprintf("tmpl/%s/%s%s.tmpl", "none", strings.ToLower(f.HttpMethod), postfix), f.TmplParams)...)
		}
	}
	err := os.WriteFile(fmt.Sprintf("%s/%s/%s", outdir, packageName, clientFilename), b, 0644)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Msg("error writing file")
	}
}

func generateBeginning(tfs fs.FS, packageName string) []byte {
	tmplParams := TmplParams{
		PackageName: packageName,
	}
	return generate(tfs, "tmpl/client.tmpl", tmplParams)
}

func generate(fs fs.FS, tmplFilename string, params TmplParams) []byte {
	tpl, err := template.ParseFS(fs, tmplFilename, "tmpl/parseJson.tmpl", "tmpl/beginning.tmpl", "tmpl/client.tmpl")
	if err != nil {
		zl.Fatal().Str("tmplFilename", tmplFilename).Str("err", err.Error()).Msg("failed to load message template")
	} else if tpl == nil {
		zl.Fatal().Str("tmplFilename", tmplFilename).Msg("template is nil. failed to load message template")
	}

	var b bytes.Buffer
	if err2 := tpl.Execute(&b, params); err2 != nil {
		zl.Fatal().Str("tmplFilename", tmplFilename).Str("err", err2.Error()).Msg("failed to generate message")
	}
	return b.Bytes()
}

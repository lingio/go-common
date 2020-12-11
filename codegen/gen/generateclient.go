package gen

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
	QueryParams  []string
}

func GenerateAll(funcs []Func, outdir string, write bool) {
	b := make([]byte, 0)
	b = append(b, generateBeginning(write)...)

	for _, f := range funcs {
		postfix := ""
		if f.TmplParams.Params == "" {
			postfix += "NoParams"
		}
		if (f.HttpMethod == "POST" || f.HttpMethod == "PUT") && f.TmplParams.BodyType == "" {
			postfix += "NoBody"
		}
		b = append(b, generate(fmt.Sprintf("tmpl/%s%s.tmpl", strings.ToLower(f.HttpMethod), postfix), f.TmplParams)...)
	}
	clientFilename := "readclient.gen.go"
	if write {
		clientFilename = "writeclient.gen.go"
	}
	err := ioutil.WriteFile(fmt.Sprintf("%s/%s", outdir, clientFilename), b, 0644)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Msg("error writing file")
	}
}

func generateBeginning(write bool) []byte {
	if write {
		return generate("tmpl/beginningWrite.tmpl", TmplParams{})
	}
	return generate("tmpl/beginningRead.tmpl", TmplParams{})
}

func generate(tmplFilename string, params TmplParams) []byte {
	tpl, err := template.ParseFiles(tmplFilename)
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

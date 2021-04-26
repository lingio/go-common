package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"text/template"

	zl "github.com/rs/zerolog/log"
)

type BucketSpec struct {
	TypeName   string
	DbTypeName string
	BucketName string
	Template   string
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

	if len(os.Args) < 2 {
		zl.Fatal().Msg("Usage: go run main.go <spec.json>")
	}
	spec := readSpec(os.Args[1])
	dir := path.Dir(os.Args[1])
	for _, b := range spec.Buckets {
		bytes := generate("tmpl/"+b.Template, TmplParams{
			ServiceName: spec.ServiceName,
			TypeName:    b.TypeName,
			DbTypeName:  b.DbTypeName,
			BucketName:  b.BucketName,
		})
		err := ioutil.WriteFile(fmt.Sprintf("%s/%s.gen.go", dir, b.BucketName), bytes, 0644)
		if err != nil {
			zl.Fatal().Str("err", err.Error()).Msg("failed to load minio template")
		}
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
	TypeName    string
	DbTypeName  string
	BucketName  string
	ServiceName string
}

type TmplParams2 struct {
	Buckets []BucketSpec
}

func generate(tmplFilename string, params interface{}) []byte {
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

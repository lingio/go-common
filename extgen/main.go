package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/lingio/go-common/extgen/gen"
	zl "github.com/rs/zerolog/log"
)

var (
	//go:embed tmpl
	templateFS embed.FS
)

/*
Example: To call Partner Service from Person Service:
1. Create a directory in Person Service for the new client
2. Create a specification file in that directory with the endpoints you need to access (see example-spec.json)
3. Run the following command:
>go run main.go ~/go/person-service/extservices/partnerclient/spec.json ~/go/partner-service/spec.yaml
*/
func main() {
	if len(os.Args) < 3 {
		zl.Fatal().Msg("Usage: go run main.go <extconfig.json> <spec.json>")
	}
	extConfig := readExtConfig(os.Args[1])
	targetDir := path.Dir(os.Args[1])
	spec := os.Args[2]

	// Copy the model.gen.go file and modify the packagename to match its new destination
	srcDir := path.Dir(os.Args[2])
	modelFile := fmt.Sprintf("%s/models/model.gen.go", srcDir)
	copyModelFile(modelFile, targetDir, path.Base(targetDir))

	gen.GenerateFromSpec(templateFS, extConfig, spec, targetDir)
	copyVersionFile(srcDir, targetDir)
}

func copyModelFile(filename string, targetDir string, packageName string) {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		zl.Fatal().Str("error", err.Error()).Msg("failed to read the models.gen.go file")
	}

	data := bytes.ReplaceAll(input, []byte("package models"), []byte(fmt.Sprintf("package %s", packageName)))

	// Some services use an older oapi-codegen that we haven't upgraded yet.
	// The generated models file will reference a deprecated pkg.
	// This provides an backwards-and-forwards-compatible upgrade path.
	// Simply run `go get github.com/oapi-codegen/runtime/types` in service dir.
	if out, err := exec.Command("go", "list", "-json").Output(); err == nil && bytes.Contains(out, []byte("github.com/oapi-codegen/runtime/types")) {
		data = bytes.ReplaceAll(data, []byte("github.com/deepmap/oapi-codegen/pkg/types"), []byte("github.com/oapi-codegen/runtime/types"))
	}

	err = ioutil.WriteFile(fmt.Sprintf("%s/model.gen.go", targetDir), data, 0644)
	if err != nil {
		zl.Fatal().Str("error", err.Error()).Msg("failed to write the models.gen.go file")
	}
}

func readExtConfig(filename string) gen.ExtSpec {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Str("filename", filename).Msg("failed to load ext spec file")
	}
	spec := gen.ExtSpec{}
	err = json.Unmarshal(file, &spec)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Str("filename", filename).Msg("failed to unmarshal storage spec file")
	}
	return spec
}

func copyVersionFile(sourceDir string, targetDir string) {
	src, err := ioutil.ReadFile(fmt.Sprintf("%s/build/version", sourceDir))
	if err != nil {
		zl.Warn().Str("err", err.Error()).Msg("failed to load version file")
		return
	}

	err = ioutil.WriteFile(fmt.Sprintf("%s/version", targetDir), src, 0644)
	if err != nil {
		zl.Warn().Str("err", err.Error()).Msg("failed to write version file")
	}
}

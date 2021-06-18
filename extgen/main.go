package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/lingio/go-common/extgen/gen"
	zl "github.com/rs/zerolog/log"
)

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

	gen.GenerateFromSpec(extConfig, spec, targetDir)
}

func copyModelFile(filename string, targetDir string, packageName string) {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		zl.Fatal().Str("error", err.Error()).Msg("failed to read the models.gen.go file")
	}

	data := []byte(strings.Replace(string(input), "package models", fmt.Sprintf("package %s", packageName), 1))
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

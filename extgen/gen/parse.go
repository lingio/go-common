package gen

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"slices"
	"strings"

	"github.com/go-yaml/yaml"
	zl "github.com/rs/zerolog/log"
)

type InParams struct {
	In          string
	Name        string
	Description string
	Required    bool
	Schema      struct {
		Type string
	}
}

type ajson struct {
	Schema struct {
		Type string `yaml:"$ref"`
	}
}

type reqBody struct {
	Content struct {
		Json ajson `yaml:"application/json"`
	}
}

type resp struct {
	Type string `yaml:"$ref"`
}

type BearerAuth struct {
	Type   string
	Scheme string
}

type ApiKeyAuth struct {
	Type string
	In   string
	Name string
}

type SecurityScheme struct {
	BearerAuth *BearerAuth
	ApiKeyAuth *ApiKeyAuth
}

type SecInfo struct {
	BearerAuth []string `yaml:"bearerAuth"`
	ApiKeyAuth []string `yaml:"apiKeyAuth"`
}

type FuncSpec struct {
	Summary     string
	OperationID string    `yaml:"operationId"`
	Security    []SecInfo `yaml:"security"`
	Parameters  []InParams
	RequestBody reqBody `yaml:"requestBody"`
	Responses   struct {
		Resp200 resp `yaml:"200"`
		Resp201 resp `yaml:"201"`
		RespErr resp `yaml:"default"`
	}
}

type Spec struct {
	Get        FuncSpec
	Put        FuncSpec
	Patch      FuncSpec
	Post       FuncSpec
	Delete     FuncSpec
	Parameters []InParams
}

func ReadSpec(filename string) map[string]Func {

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		zl.Fatal().Err(err).Str("filename", filename).Msg("failed to read spec")
	}

	s := string(b)
	rows := strings.Split(s, "\n")
	inPaths := false
	endpointStrs := make([]string, 0) // each entry is a string of one endpoint
	funcStr := ""
	paths := make([]string, 0)
	for _, row := range rows {
		if strings.HasPrefix(row, "paths") {
			inPaths = true
			continue
		}
		trimmedRow := strings.TrimSpace(row)
		if inPaths && (strings.HasPrefix(trimmedRow, "/") || strings.HasPrefix(trimmedRow, "'/") || strings.HasPrefix(trimmedRow, "\"/")) {
			p := strings.TrimSuffix(trimmedRow, ":")
			paths = append(paths, p)
			if funcStr != "" {
				endpointStrs = append(endpointStrs, funcStr)
			}
			funcStr = ""
			continue
		}
		if inPaths && row != "" && !strings.HasPrefix(row, " ") {
			endpointStrs = append(endpointStrs, funcStr)
			break
		}
		if inPaths && trimmedRow != "" {
			funcStr += row + "\n"
		}
	}

	funcs := make(map[string]Func, 0)
	for i, eps := range endpointStrs { // one entry for each endpoint (independent upon HTTP Method)
		path := paths[i]
		spec := Spec{}
		err = yaml.Unmarshal([]byte(eps), &spec)
		if err != nil {
			zl.Fatal().Err(err).Str("filename", filename).Msg("failed to parse spec")
		}

		if spec.Get.OperationID != "" {
			funcs[spec.Get.OperationID] = Func{
				TmplParams: templParams(path, spec.Parameters, spec.Get),
				HttpMethod: "GET",
			}
		}
		if spec.Put.OperationID != "" {
			funcs[spec.Put.OperationID] = Func{
				TmplParams: templParams(path, spec.Parameters, spec.Put),
				HttpMethod: "PUT",
			}
		}
		if spec.Patch.OperationID != "" {
			funcs[spec.Patch.OperationID] = Func{
				TmplParams: templParams(path, spec.Parameters, spec.Patch),
				HttpMethod: "PATCH",
			}
		}
		if spec.Post.OperationID != "" {
			funcs[spec.Post.OperationID] = Func{
				TmplParams: templParams(path, spec.Parameters, spec.Post),
				HttpMethod: "POST",
			}
		}
		if spec.Delete.OperationID != "" {
			funcs[spec.Delete.OperationID] = Func{
				TmplParams: templParams(path, spec.Parameters, spec.Delete),
				HttpMethod: "Delete",
			}
		}
	}
	return funcs
}

type QueryParam struct {
	Name string
	Type string
}

func templParams(path string, inheritedParams []InParams, fs FuncSpec) TmplParams {
	params := ""
	params2 := ""
	queryParams := make([]QueryParam, 0)
	numPathParams := 0
	numQueryParams := 0
	for _, p := range slices.Concat(inheritedParams, fs.Parameters) {
		if idx := strings.LastIndex(p.Name, "Id"); idx > 0 {
			p.Name = p.Name[0:idx] + "ID"
		}

		if p.In == "path" {
			numPathParams += 1
			if numPathParams > 1 {
				params += ", "
				params2 += ", "
			}
			params += p.Name + " " + gotype(p.Schema.Type)
			params2 += p.Name
		} else if p.In == "query" {
			numQueryParams += 1
			if numPathParams+numQueryParams > 1 {
				params += ", "
			}
			params += p.Name + " *" + gotype(p.Schema.Type)
			queryParams = append(queryParams, QueryParam{
				Name: p.Name,
				Type: p.Schema.Type,
			})
		} else {
			zl.Fatal().Str("parameters.in", p.In).Msg("unexpected value for parameter type")
		}
	}

	/*
		if hasTokenAuth(fs) {
			if params == "" {
				params += "token string"
			} else {
				params += ", token string"
			}
		}
	*/

	rt := fs.Responses.Resp200.Type
	if fs.Responses.Resp201.Type != "" {
		rt = fs.Responses.Resp201.Type
	}

	if fs.OperationID == "SendHubspotSupportEmail" {
		fmt.Println("params: ", params)
		fmt.Println("params2:", params2)
		fmt.Println(fs)
	}

	return TmplParams{
		Path:         path,
		PathTemplate: templetize(path),
		FuncName:     fs.OperationID,
		BodyType:     lastPart(fs.RequestBody.Content.Json.Schema.Type),
		RetObjType:   lastPart(rt),
		Params:       params,
		Params2:      params2,
		QueryParams:  queryParams,
		TokenAuth:    hasTokenAuth(fs),
		ApiKeyAuth:   hasApiKeyAuth(fs),
	}
}

func hasTokenAuth(fs FuncSpec) bool {
	for _, s := range fs.Security {
		if s.BearerAuth != nil {
			return true
		}
	}
	return false
}

func hasApiKeyAuth(fs FuncSpec) bool {
	for _, s := range fs.Security {
		if s.ApiKeyAuth != nil {
			return true
		}
	}
	return false
}

func templetize(path string) string {
	r := regexp.MustCompile(`{.*?}`)
	return r.ReplaceAllString(path, "%s")
}

func lastPart(s string) string {
	strs := strings.Split(s, "/")
	return strs[len(strs)-1]
}

func gotype(s string) string {
	if s == "boolean" {
		return "bool"
	}
	return s
}

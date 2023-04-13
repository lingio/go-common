package common

import (
	"os"
	"strings"
)

type Environment string

var (
	EnvUnknown    Environment = "unknown"
	EnvDevelop    Environment = "develop"
	EnvStaging    Environment = "staging"
	EnvProduction Environment = "production"
)

type Env struct {
	EnvName    string
	ProjectID  string
	ConfigFile string
}

func SetupEnv() *Env {
	env := os.Getenv("ENV")

	e := &Env{
		EnvName:    env,
		ProjectID:  "lingio-stage",
		ConfigFile: "local",
	}

	// stage, stage-glesys, stage-gcp
	if strings.HasPrefix(env, "stage") {
		e.ProjectID = "lingio-stage"
		e.ConfigFile = "stage"
	}

	// prod, production, production-glesys
	if strings.HasPrefix(env, "prod") {
		e.ProjectID = "lingio-prod"
		e.ConfigFile = "production"
	}
	return e
}

func ParseEnv() Environment {
	env := os.Getenv("ENV")
	if env == "prod" || env == "production" {
		return EnvProduction
	}
	if env == "stage" || env == "staging" {
		return EnvStaging
	}
	if strings.HasPrefix(env, "local") {
		return EnvDevelop
	}
	return EnvUnknown
}

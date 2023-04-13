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
	switch ParseEnv() {
	case EnvDevelop:
		return &Env{
			EnvName:    env,
			ProjectID:  "lingio-stage",
			ConfigFile: env,
		}
	case EnvStaging:
		return &Env{
			EnvName:    env,
			ProjectID:  "lingio-stage",
			ConfigFile: env,
		}
	case EnvProduction:
		return &Env{
			EnvName:    env,
			ProjectID:  "lingio-prod",
			ConfigFile: env,
		}
	}
	panic("SetupEnv: unknown env: " + env)
}

func ParseEnv() Environment {
	env := os.Getenv("ENV")

	// prod, production, production-glesys
	if strings.HasPrefix(env, "prod") {
		return EnvProduction
	}

	// stage, stage-glesys, stage-gcp
	if strings.HasPrefix(env, "stage") {
		return EnvStaging
	}

	if strings.HasPrefix(env, "local") {
		return EnvDevelop
	}
	return EnvUnknown
}

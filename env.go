package common

import (
	"strings"

	zl "github.com/rs/zerolog/log"
)

type Environment string

var (
	EnvUnknown    Environment = "unknown"
	EnvDevelop    Environment = "develop"
	EnvStaging    Environment = "staging"
	EnvProduction Environment = "production"
)

type Env struct {
	EnvName string
	// GCP project ID.
	ProjectID  string
	ConfigFile string
	Environment
}

func SetupEnv() *Env {
	return env
}

func setupEnv() *Env {
	envstr := MustGetenv("ENV")
	switch env := ParseEnv(envstr); env {
	case EnvDevelop:
		return &Env{
			EnvName:     envstr,
			ProjectID:   "lingio-stage",
			ConfigFile:  envstr,
			Environment: env,
		}
	case EnvStaging:
		return &Env{
			EnvName:     envstr,
			ProjectID:   "lingio-stage",
			ConfigFile:  envstr,
			Environment: env,
		}
	case EnvProduction:
		return &Env{
			EnvName:     envstr,
			ProjectID:   "lingio-prod",
			ConfigFile:  envstr,
			Environment: env,
		}
	}
	zl.Fatal().Msgf("setupEnv: unknown env %q", envstr)
	return nil
}

func ParseEnv(env string) Environment {
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

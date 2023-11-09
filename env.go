package common

import (
	"fmt"
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
	EnvName string
	// GCP project ID.
	ProjectID  string
	ConfigFile string
	Environment
}

func SetupEnv(envstr string) (*Env, error) {
	switch env := ParseEnv(envstr); env {
	case EnvDevelop:
		return &Env{
			EnvName:     envstr,
			ProjectID:   "lingio-stage",
			ConfigFile:  envstr,
			Environment: env,
		}, nil
	case EnvStaging:
		return &Env{
			EnvName:     envstr,
			ProjectID:   "lingio-stage",
			ConfigFile:  envstr,
			Environment: env,
		}, nil
	case EnvProduction:
		return &Env{
			EnvName:     envstr,
			ProjectID:   "lingio-prod",
			ConfigFile:  envstr,
			Environment: env,
		}, nil
	default:
		return nil, fmt.Errorf("SetupEnv: unknown env %q must have either prod|stage|local prefix", env)
	}
	/* unreachable */
	// return Env{}, nil
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

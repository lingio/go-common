package common

import "os"

type Env struct {
	EnvName     string
	ProjectID   string
	ConfigFile  string
}

func SetupEnv() *Env {
	env := os.Getenv("ENV")

	e := &Env{
		EnvName:     env,
		ProjectID:   "lingio-stage",
		ConfigFile:  "local",
	}

	if env == "stage" {
		e.ProjectID = "lingio-stage"
		e.ConfigFile = "stage"
	}
	if env == "prod" || env == "production" {
		e.ProjectID = "lingio-prod"
		e.ConfigFile = "production"
	}
	return e
}

package common

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	env *Env = setupEnv()
)

func init() {
	// setup uniform logging fields asap so we get them everywhere
	zerolog.LevelFieldName = "severity"
	zerolog.TimestampFieldName = "timestamp"
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// replace the default logger
	log.Logger = setupZerologger()
}

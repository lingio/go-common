package common

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// setup uniform logging fields asap so we get them everywhere
	zerolog.LevelFieldName = "severity"
	zerolog.TimestampFieldName = "timestamp"
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// replace the default logger

	switch ParseEnv(os.Getenv("ENV")) {
	case EnvDevelop, EnvUnknown:
		log.Logger = zerolog.New(zerolog.NewConsoleWriter(
			func(w *zerolog.ConsoleWriter) {
				// basically, only log message, error and full_trace
				w.FieldsExclude = []string{
					"host", "remote_ip", "user_agent", "protocol", "method", "httpRequest",
					"uri", "status", "latency_us", "latency_human", "bytes_in", "bytes_out",
					"logging.googleapis.com/spanId", "logging.googleapis.com/trace_sampled",
					"logging.googleapis.com/operation", "correlation_id", "path",
					"logging.googleapis.com/trace", "trace",
				}
			},
		)).With().Timestamp().Logger()
	default:
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
}

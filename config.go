package common

import (
	"os"

	zl "github.com/rs/zerolog/log"
)

// MinioConfig describes the required fields to setup a s3 minio client.
type MinioConfig struct {
	Host        string
	AccessKeyID string
	SSL         bool `json:"SSL"`
}

// RedisConfig describes connectivity options for setting up a redis client
// using the included SetupRedisClient func in this pkg.
type RedisConfig struct {
	Addr             string // for testing locally using one redis server
	MasterName       string // sentinel master
	ServiceDNS       string // lookup sentinel servers on this domain name
	SentinelPassword *string
	MasterPassword   *string
}

type MonitorConfig struct {
	TempoHost  string
	CloudTrace CloudTraceConfig
}

type CloudTraceConfig struct {
	ProjectID string
	Enabled   bool
}

func MustGetenv(key string) string {
	val, varok := os.LookupEnv(key)
	if !varok {
		zl.Fatal().Msg("missing env. variable " + key)
		// unreachable!
	}
	return val
}

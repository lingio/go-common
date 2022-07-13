module github.com/lingio/go-common

go 1.14

require (
	github.com/deepmap/oapi-codegen v1.10.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/getkin/kin-openapi v0.94.0
	github.com/go-redis/redis/v8 v8.11.4
	github.com/go-redsync/redsync/v4 v4.4.2
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/labstack/echo-contrib v0.9.0
	github.com/labstack/echo/v4 v4.7.2
	github.com/minio/minio-go/v7 v7.0.15
	github.com/rs/zerolog v1.15.0
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.32.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.33.0
	go.opentelemetry.io/otel v1.8.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.8.0
	go.opentelemetry.io/otel/sdk v1.8.0
	go.opentelemetry.io/otel/trace v1.8.0
	golang.org/x/time v0.0.0-20220411224347-583f2d630306
)

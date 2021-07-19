package common

import (
	"context"
	"fmt"
	"github.com/deepmap/oapi-codegen/pkg/middleware"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	zl "github.com/rs/zerolog/log"
	"github.com/ziflex/lecho/v2"
	"os"
)

type ErrorStruct struct {
	Message string `json:"message"`
}

func NewEchoServerWithLingioStdConfig(swagger *openapi3.Swagger) *echo.Echo {
	e := echo.New()
	lechologger := lecho.New(os.Stdout, lecho.WithTimestamp(), lecho.WithCaller())
	e.Use(lecho.Middleware(lecho.Config{Logger: lechologger})) // log all requests
	e.Use(echomiddleware.BodyLimit("1M"))                      // limit request body size
	e.Use(echomiddleware.CORS())
	//e.Use(echomiddleware.Gzip())

	// Set up request validation
	options := &middleware.Options{Options: *openapi3filter.DefaultOptions}
	options.Options.AuthenticationFunc = func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		return nil
	}
	//e.Use(oapiRequestValidatorWithOptions(swagger, options)) // check all requests against the OpenAPI schema
	options.Skipper = metricsSkipper
	e.Use(middleware.OapiRequestValidatorWithOptions(swagger, options)) // check all requests against the OpenAPI schema

	// Init Prometheus
	p := prometheus.NewPrometheus("echo", nil)
	p.Use(e)

	return e
}

func metricsSkipper(ectx echo.Context) bool {
	return ectx.Path() == "/metrics"
}

func Respond(ctx echo.Context, statusCode int, val interface{}, etag string) error {
	if etag != "" {
		ctx.Response().Header().Set("Cache-Control", "must-revalidate")
		ctx.Response().Header().Set("etag", etag)
	} else {
		ctx.Response().Header().Set("Pragma", "no-cache")
		ctx.Response().Header().Set("Cache-Control", "no-store")
		ctx.Response().Header().Set("max-age", "0")
	}
	if val != nil {
		return ctx.JSON(statusCode, val)
	}
	return ctx.NoContent(statusCode)
}

func RespondFile(ctx echo.Context, statusCode int, file []byte, fileName string, contentType string, etag string) error {
	if etag != "" {
		ctx.Response().Header().Set("Cache-Control", "must-revalidate")
		ctx.Response().Header().Set("etag", etag)
	} else {
		ctx.Response().Header().Set("Pragma", "no-cache")
		ctx.Response().Header().Set("Cache-Control", "no-store")
		ctx.Response().Header().Set("max-age", "0")
	}
	ctx.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s;", fileName))
	return ctx.Blob(statusCode, contentType, file)
}

func RespondError(ctx echo.Context, le *Error) error {
	// Log error
	zle := zl.Warn()
	if le.HttpStatusCode >= 500 {
		zle = zl.Error().Err(le)
	}
	zle.Int("httpStatusCode", le.HttpStatusCode)
	zle.Str("trace", le.Trace)
	for k, v := range le.Map {
		zle = zle.Str(k, v)
	}
	zle.Msg(le.Message)

	// Create and set error object on the Echo Context
	e := ErrorStruct{
		Message: le.Message,
	}
	return Respond(ctx, le.HttpStatusCode, e, "")
}

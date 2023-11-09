package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/deepmap/oapi-codegen/pkg/middleware"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	zl "github.com/rs/zerolog/log"
)

type ErrorStruct struct {
	Message string `json:"message"`
}

type EchoConfig struct {
	BodyLimit echo.MiddlewareFunc

	// Initializes per-request logger. Defaults to GCP-style if nil.
	RequestLogger func(http.ResponseWriter) zerolog.Logger

	// Hard to autodetect where server is running so programmer's responsibility
	// specify request log output format. Defaults to GCP-style if nil.
	RequestLogFormatter RequestLogFormatter
}

var DefaultEchoConfig = EchoConfig{
	BodyLimit: echomiddleware.BodyLimit("1M"),
}

func combineSkippers(skippers ...echomiddleware.Skipper) echomiddleware.Skipper {
	return func(ctx echo.Context) bool {
		for _, skipper := range skippers {
			if skipper(ctx) {
				return true
			}
		}
		return false
	}
}

func NewEchoServerWithConfig(env *Env, swagger *openapi3.T, config EchoConfig) *echo.Echo {
	if config.RequestLogFormatter == nil {
		config.RequestLogFormatter = env.gcpRequestLogFormatter
	}
	if config.RequestLogger == nil {
		config.RequestLogger = gcpRequestLogger
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Init Prometheus
	p := prometheus.NewPrometheus("echo", nil)
	isMetricRequest := func(ctx echo.Context) bool {
		return ctx.Path() == p.MetricsPath
	}
	isOpsRequest := func(ctx echo.Context) bool {
		return strings.HasPrefix(ctx.Path(), "/ops") || strings.HasPrefix(ctx.Path(), "/debug")
	}
	devopsRequestSkipper := combineSkippers(isMetricRequest, isOpsRequest)

	// Set up a basic Echo router and its middlewares
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var echoError *echo.HTTPError
		if errors.As(err, &echoError) {
			e.DefaultHTTPErrorHandler(echoError, c)
		} else {
			e.DefaultHTTPErrorHandler(err, c)
		}
	}

	e.Use(echomiddleware.RequestID())

	e.Use(otelecho.Middleware(
		swagger.Info.Title,
		otelecho.WithSkipper(devopsRequestSkipper),
		otelecho.WithTracerProvider(otel.GetTracerProvider()),
	))

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqlog := config.RequestLogger(c.Response().Writer)
			originalCtx := c.Request().Context()
			wrappedCtx := reqlog.WithContext(originalCtx)
			c.SetRequest(c.Request().WithContext(wrappedCtx))
			return next(c)
		}
	})
	e.Use(echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		LogURI:           true,
		LogStatus:        true,
		LogLatency:       true,
		LogRemoteIP:      true,
		LogHost:          true,
		LogError:         true,
		LogMethod:        true,
		LogProtocol:      true,
		LogResponseSize:  true,
		LogContentLength: true,
		LogRoutePath:     true,
		LogUserAgent:     true,
		LogValuesFunc:    config.RequestLogFormatter,
	}))
	e.Use(config.BodyLimit) // limit request body size
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		Skipper:      echomiddleware.DefaultSkipper,
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		// NOTE: Max-Age is constrained by browser implementation, so the actual max age might be less.
		MaxAge: int((30 * 24 * time.Hour).Seconds()),
	}))
	e.Use(echomiddleware.GzipWithConfig(echomiddleware.GzipConfig{
		Skipper: devopsRequestSkipper,
	}))

	p.Use(e) // add prometheus /metrics endpoint

	// Set up request validation
	e.Use(middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
		Skipper: devopsRequestSkipper,
	}))

	return e
}

func NewEchoServer(env *Env, swagger *openapi3.T) *echo.Echo {
	return NewEchoServerWithConfig(env, swagger, DefaultEchoConfig)
}

type GracefulServer interface {
	Start(addr string) error
	Shutdown(context.Context) error
}

var DefaultServeSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func ServeUntilSignal(e GracefulServer, addr string, signals ...os.Signal) {
	zl.Info().Str("addr", addr).Msg("starting api server")

	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			zl.Warn().Str("error", err.Error()).Msg("fatal error serving api")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, signals...)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	zl.Info().Msg("shutting down api server")
	if err := e.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		zl.Warn().Err(err).Msg("error shutting down api server")
	}

	// Best effort cleanup on service shutdown.
	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		zl.Info().Msg("flushing traces")
		if err := tp.ForceFlush(ctx); err != nil {
			zl.Warn().Err(err).Msg("error flusing trace provider")
		}
		if err := tp.Shutdown(ctx); err != nil {
			zl.Warn().Err(err).Msg("error shuting down trace provider")
		}
	}
}

func ShutdownServices(services ...interface{ Shutdown(context.Context) error }) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, svc := range services {
		if err := svc.Shutdown(ctx); err != nil {
			zl.Warn().Err(err).Msg("error shutting down internal service")
		}
	}
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

// RespondError is deprecated. Return an error directly in the middleware instead.
func RespondError(ctx echo.Context, le *Error) error {
	// returning le directly will busy loop somewhere in echo
	return Errorf(le)
}

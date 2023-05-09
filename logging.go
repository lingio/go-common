package common

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

func InitLogging() {
	zerolog.LevelFieldName = "severity"
	zerolog.TimestampFieldName = "timestamp"
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

type RequestLogFormatter func(c echo.Context, v echomiddleware.RequestLoggerValues) error
type RequestLogger func(http.ResponseWriter) zerolog.Logger

func gcpRequestLogFormatter(c echo.Context, v echomiddleware.RequestLoggerValues) error {
	var (
		spanCtx = trace.SpanFromContext(c.Request().Context()).SpanContext()

		// log level info if v.Error is nil, otherwise error
		zle = zerolog.Ctx(c.Request().Context()).Err(v.Error)
	)

	zle.
		// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#HttpRequest
		Dict("httpRequest",
			zerolog.Dict().
				Str("requestMethod", v.Method).
				Str("requestUrl", v.URI).
				Str("requestSize", v.ContentLength). // sent by client
				Int("status", v.Status).
				Str("responseSize", strconv.FormatInt(v.ResponseSize, 10)).
				Str("userAgent", v.UserAgent).
				Str("remoteIp", v.RemoteIP).
				// Str("serverIp", v.ServerIP) // does not exit
				Str("referer", v.Referer).
				Str("latency", v.Latency.String()).
				// Bool("cacheLookup", false) // does not exit
				// Bool("cacheHit", false) // does not exit
				// Bool("cacheValidatedWithOriginServer", false) // does not exit
				// Str("cacheFillBytes", "") // does not exit
				Str("protocol", v.Protocol),
		).
		Str("path", v.RoutePath). // /users/:userid
		Str("logging.googleapis.com/trace", "/projects/lingio-stage/traces/"+spanCtx.TraceID().String()).
		Str("logging.googleapis.com/spanId", spanCtx.SpanID().String()).
		Bool("logging.googleapis.com/trace_sampled ", spanCtx.IsSampled())

	// https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
	//
	// ... If your log entry contains an exception stack trace, the exception
	// stack trace should be set in this message JSON log field, ...
	//
	if v.Error != nil {
		zle.Msg(FullErrorTrace(v.Error))
	} else {
		zle.Msg("request") // actually log it
	}
	return nil
}

func defaultRequestLogFormatter(c echo.Context, v echomiddleware.RequestLoggerValues) error {
	var (
		span = trace.SpanFromContext(c.Request().Context())

		// log level info if v.Error is nil, otherwise error
		zle = zerolog.Ctx(c.Request().Context()).Err(v.Error)
	)

	zle.Str("host", v.Host).
		Str("remote_ip", v.RemoteIP).
		Str("user_agent", v.UserAgent).
		Str("protocol", v.Protocol).
		Str("method", v.Method).
		Str("uri", v.URI).        // /users/5?q=1
		Str("path", v.RoutePath). // /users/:userid
		Int("status", v.Status).
		Int64("latency_us", v.Latency.Microseconds()).
		Str("latency_human", v.Latency.String()).
		Str("bytes_in", v.ContentLength).
		Int64("bytes_out", v.ResponseSize).
		Str("trace_id", span.SpanContext().TraceID().String())

	if v.Error != nil {
		zle.Str("full_trace", FullErrorTrace(v.Error))
	}

	zle.Msg("request") // actually log it

	return nil
}

var _ RequestLogFormatter = gcpRequestLogFormatter
var _ RequestLogFormatter = defaultRequestLogFormatter

func gcpRequestLogger(w http.ResponseWriter) zerolog.Logger {
	return zerolog.New(os.Stderr).With().
		Timestamp().
		Dict("logging.googleapis.com/operation",
			zerolog.Dict().
				Str("id", w.Header().Get(echo.HeaderXRequestID)),
		).
		Logger()

}

func defaultRequestLog(w http.ResponseWriter) zerolog.Logger {
	return zerolog.New(os.Stderr).With().
		Timestamp().
		Str("correlation_id", w.Header().Get(echo.HeaderXRequestID)).
		Logger()

}

var _ RequestLogger = gcpRequestLogger

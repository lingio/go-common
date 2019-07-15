package log

import (
	"context"
	"fmt"
	golog "log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/lingio/go-common/logicerr"

	googlelog "cloud.google.com/go/logging"

	"go.opencensus.io/trace"
)

// LingioLogger represents a logger used to log events
type LingioLogger struct {
	env         string
	projectID   string
	serviceName string

	loggers map[googlelog.Severity]*golog.Logger

	parentLogger *googlelog.Logger
	entryLogger  *googlelog.Logger
	client       *googlelog.Client
}

// NewLingioLogger creates a new LingioLogger that can be used for logging
func NewLingioLogger(env string, projectID string, serviceName string) *LingioLogger {
	logger := LingioLogger{}
	logger.env = env
	logger.projectID = projectID
	logger.serviceName = serviceName

	logger.loggers = make(map[googlelog.Severity]*golog.Logger)

	switch env {
	case "local":
		// Create stdout loggers
		logger.loggers[googlelog.Error] = golog.New(os.Stderr, "Error: ", golog.Lshortfile)
		logger.loggers[googlelog.Warning] = golog.New(os.Stderr, "Warning: ", golog.Lshortfile)
		logger.loggers[googlelog.Info] = golog.New(os.Stderr, "Info: ", golog.Lshortfile)
		logger.loggers[googlelog.Debug] = golog.New(os.Stderr, "Debug: ", golog.Lshortfile)
	default:
		// Create goolge stackdriver loggers
		ctx := context.Background()
		client, err := googlelog.NewClient(ctx, projectID)
		if err != nil {
			fmt.Printf("Failed to initialize logger: %s", err)
			os.Exit(1)
		}
		if err := client.Ping(ctx); err != nil {
			fmt.Printf("Failed to connect to logger: %s", err)
			os.Exit(1)
		}

		m := make(map[string]string)
		m["env"] = logger.env
		m["projectID"] = logger.projectID
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Printf("Error when trying to get hostname: %v", err.Error())
			hostname = fmt.Sprintf("got-no-hostname-%x", rand.Uint64())
		}
		m["machine"] = fmt.Sprintf("go-%v@%s", os.Getpid(), hostname)

		logger.client = client
		parentLogger := client.Logger(serviceName, googlelog.CommonLabels(m))
		logger.parentLogger = parentLogger
		logger.entryLogger = client.Logger(serviceName+"_entries", googlelog.CommonLabels(m))

		logger.loggers[googlelog.Error] = parentLogger.StandardLogger(googlelog.Error)
		logger.loggers[googlelog.Warning] = parentLogger.StandardLogger(googlelog.Warning)
		logger.loggers[googlelog.Info] = parentLogger.StandardLogger(googlelog.Info)
		logger.loggers[googlelog.Debug] = parentLogger.StandardLogger(googlelog.Debug)
	}

	return &logger
}

// Debug logs a debug message
func (ll *LingioLogger) Debug(ctx context.Context, message string, m map[string]string) {
	ll.logm(ctx, message, googlelog.Debug, m)
}

// DebugUser logs a debug message
func (ll *LingioLogger) DebugUser(ctx context.Context, message string, partnerID string, userID string, m map[string]string) {
	m = makeUserMapFromExsisting(partnerID, userID, m)
	ll.logm(ctx, message, googlelog.Debug, m)
}

// Info logs an info message
func (ll *LingioLogger) Info(ctx context.Context, message string, m map[string]string) {
	ll.logm(ctx, message, googlelog.Info, m)
}

// InfoUser logs an info message
func (ll *LingioLogger) InfoUser(ctx context.Context, message string, partnerID string, userID string, m map[string]string) {
	m = makeUserMapFromExsisting(partnerID, userID, m)
	ll.logm(ctx, message, googlelog.Info, m)
}

// Warning logs a warning message
func (ll *LingioLogger) Warning(ctx context.Context, message string, m map[string]string) {
	ll.logm(ctx, message, googlelog.Warning, m)
}

// WarningUser logs a warning message
func (ll *LingioLogger) WarningUser(ctx context.Context, message string, partnerID string, userID string, m map[string]string) {
	m = makeUserMapFromExsisting(partnerID, userID, m)
	ll.logm(ctx, message, googlelog.Warning, m)
}

// WarningE logs a logicerr.Error warning with a custom message
func (ll *LingioLogger) WarningE(ctx context.Context, message string, e *logicerr.Error) {
	m := e.InfoMap
	if m == nil {
		m = make(map[string]string)
	}
	m["error_code"] = fmt.Sprintf("%v", e.HTTPStatusCode)
	m["trace"] = e.Trace
	m["error_message"] = e.Message
	ll.logm(ctx, message, googlelog.Warning, m)
}

// WarningUserE logs a warning message
func (ll *LingioLogger) WarningUserE(ctx context.Context, message string, e *logicerr.Error, partnerID string, userID string) {
	m := makeUserMapFromExsisting(partnerID, userID, e.InfoMap)
	m["error_code"] = fmt.Sprintf("%v", e.HTTPStatusCode)
	m["trace"] = e.Trace
	m["error_message"] = e.Message
	ll.logm(ctx, message, googlelog.Warning, m)
}

// Error logs an error message
func (ll *LingioLogger) Error(ctx context.Context, message string, m map[string]string) {
	ll.logm(ctx, message, googlelog.Error, m)
}

// ErrorUser logs an error message
func (ll *LingioLogger) ErrorUser(ctx context.Context, message string, partnerID string, userID string, m map[string]string) {
	m = makeUserMapFromExsisting(partnerID, userID, m)
	ll.logm(ctx, message, googlelog.Error, m)
}

// ErrorE logs a logicerr.Error error with a custom message
func (ll *LingioLogger) ErrorE(ctx context.Context, message string, e *logicerr.Error) {
	m := e.InfoMap
	if m == nil {
		m = make(map[string]string)
	}
	m["error_code"] = fmt.Sprintf("%v", e.HTTPStatusCode)
	m["trace"] = e.Trace
	m["error_message"] = e.Message
	ll.logm(ctx, message, googlelog.Error, m)
}

// ErrorUserE logs a logicerr.Error error
func (ll *LingioLogger) ErrorUserE(ctx context.Context, message string, e *logicerr.Error, partnerID string, userID string) {
	m := makeUserMapFromExsisting(partnerID, userID, e.InfoMap)
	m["error_code"] = fmt.Sprintf("%v", e.HTTPStatusCode)
	m["trace"] = e.Trace
	m["error_message"] = e.Message
	ll.logm(ctx, message, googlelog.Error, m)
}

func (ll *LingioLogger) LogEndOfHTTPRequest(ctx context.Context, message string, error_code int, request *http.Request, latency time.Duration) {
	googleRequest := &googlelog.HTTPRequest{Request: request, Latency: latency, Status: error_code}
	m := make(map[string]string)
	m["message"] = message
	spanID := ""
	traceID := ""
	span := trace.FromContext(ctx)
	if span != nil {
		spanContext := span.SpanContext()
		if spanContext.IsSampled() {
			spanID = spanContext.SpanID.String()
			traceID = spanContext.TraceID.String()
		}
	}
	// For now we decide the log level from the error_code
	severity := googlelog.Info
	if error_code >= 500 {
		severity = googlelog.Error
	} else if error_code >= 400 {
		severity = googlelog.Warning
	}
	logEntry := googlelog.Entry{Payload: m, Severity: severity, HTTPRequest: googleRequest, SpanID: spanID, Trace: traceID}
	if ll.parentLogger != nil {
		// Here we use the stackdriver logger
		ll.parentLogger.Log(logEntry)
	} else {
		logger, ok := ll.loggers[severity]
		if ok == false {
			fmt.Printf("Cannot log with this severity!! %v", severity)
			return
		}
		// These won't be set unless we do it here
		m["env"] = ll.env
		m["projectID"] = ll.projectID
		// We send 2 as the stackdepth here to that we get the right filename in the output
		_ = logger.Output(2, fmt.Sprintf("%v\n\t%v\n\t%v", message, m, logEntry))
	}
}

func makeUserMap(partnerID string, userID string) map[string]string {
	m := make(map[string]string)
	m["partnerID"] = partnerID
	m["userID"] = userID
	return m
}

func makeUserMapFromExsisting(partnerID string, userID string, m map[string]string) map[string]string {
	if m == nil {
		m = make(map[string]string)
	}
	m["partnerID"] = partnerID
	m["userID"] = userID
	return m
}

func (ll *LingioLogger) logm(ctx context.Context, message string, severity googlelog.Severity, m map[string]string) {
	if m == nil {
		m = make(map[string]string)
	}

	m["message"] = message

	// Try to get a trace from the context and if it is sampled we correlate this log with that trace
	spanID := ""
	traceID := ""
	span := trace.FromContext(ctx)
	if span != nil {
		spanContext := span.SpanContext()
		if spanContext.IsSampled() {
			spanID = spanContext.SpanID.String()
			traceID = spanContext.TraceID.String()
		}
	}

	if ll.parentLogger != nil {
		// Here we use the stackdriver logger
		ll.parentLogger.Log(googlelog.Entry{Payload: m, Severity: severity, SpanID: spanID, Trace: traceID})
	} else {
		// Here we use the local logger
		logger, ok := ll.loggers[severity]
		if ok == false {
			fmt.Printf("Cannot log with this severity!! %v", severity)
			return
		}

		// These won't be set unless we do it here
		m["env"] = ll.env
		m["projectID"] = ll.projectID

		// We send 3 as the stackdepth here to that we get the right filename in the output
		_ = logger.Output(3, fmt.Sprintf("%v\n\t%v", message, m))
	}
}

// Flush flushes the stackdriver logger
func (ll *LingioLogger) Flush() {
	if ll.client != nil {
		err := ll.entryLogger.Flush()
		if err != nil {
			fmt.Printf("Failed flushing the stackdriver entry logger: %v", err)
		}
		err = ll.parentLogger.Flush()
		if err != nil {
			fmt.Printf("Failed flushing the stackdriver parent logger: %v", err)
		}
	}
}

// Shutdown shuts down this loggers potential connection to stackdriver
func (ll *LingioLogger) Shutdown() {
	if ll.client != nil {
		if err := ll.client.Close(); err != nil {
			// TODO: We might want to send some signal to the google cloud server that this happened
			fmt.Printf("Failed to close client: %v", err)
		}
	}
}

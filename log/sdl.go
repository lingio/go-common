package log

import (
	"context"
	"fmt"
	golog "log"
	"os"

	"github.com/lingio/go-common/logicerr"

	googlelog "cloud.google.com/go/logging"
)

// LingioLogger represents a logger used to log events
type LingioLogger struct {
	env         string
	projectID   string
	serviceName string

	loggers map[googlelog.Severity]*golog.Logger

	sdlogger *googlelog.Logger
	client   *googlelog.Client
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
		sdl := client.Logger(serviceName)

		logger.sdlogger = sdl
		logger.client = client

		logger.loggers[googlelog.Error] = sdl.StandardLogger(googlelog.Error)
		logger.loggers[googlelog.Warning] = sdl.StandardLogger(googlelog.Warning)
		logger.loggers[googlelog.Info] = sdl.StandardLogger(googlelog.Info)
		logger.loggers[googlelog.Debug] = sdl.StandardLogger(googlelog.Debug)
	}

	return &logger
}

// Debug logs a debug message
func (ll *LingioLogger) Debug(message string) {
	ll.logm(message, googlelog.Debug, make(map[string]string))
}

// DebugUser logs a debug message
func (ll *LingioLogger) DebugUser(message string, partnerID string, userID string) {
	ll.logm(message, googlelog.Debug, makeUserMap(partnerID, userID))
}

// DebugUserM logs a debug message
func (ll *LingioLogger) DebugUserM(message string, partnerID string, userID string, m map[string]string) {
	m = makeUserMapFromExsisting(partnerID, userID, m)
	ll.logm(message, googlelog.Debug, m)
}

// DebugM logs a debug message
func (ll *LingioLogger) DebugM(message string, m map[string]string) {
	ll.logm(message, googlelog.Debug, m)
}

// Info logs an info message
func (ll *LingioLogger) Info(message string) {
	ll.logm(message, googlelog.Info, make(map[string]string))
}

// InfoUser logs an info message
func (ll *LingioLogger) InfoUser(message string, partnerID string, userID string) {
	ll.logm(message, googlelog.Info, makeUserMap(partnerID, userID))
}

// InfoUserM logs an info message
func (ll *LingioLogger) InfoUserM(message string, partnerID string, userID string, m map[string]string) {
	m = makeUserMapFromExsisting(partnerID, userID, m)
	ll.logm(message, googlelog.Info, m)
}

// InfoM logs an info message
func (ll *LingioLogger) InfoM(message string, m map[string]string) {
	ll.logm(message, googlelog.Info, m)
}

// Warning logs a warning message
func (ll *LingioLogger) Warning(message string) {
	ll.logm(message, googlelog.Warning, make(map[string]string))
}

// WarningE logs a warning message
func (ll *LingioLogger) WarningE(err *logicerr.Error) {
	m := err.InfoMap
	if m == nil {
		m = make(map[string]string)
	}
	m["error_code"] = fmt.Sprintf("%v", err.HTTPStatusCode)
	m["trace"] = err.Trace
	ll.logm(err.Message, googlelog.Warning, m)
}

// WarningUser logs a warning message
func (ll *LingioLogger) WarningUser(message string, partnerID string, userID string) {
	ll.logm(message, googlelog.Warning, makeUserMap(partnerID, userID))
}

// WarningUserE logs a warning message
func (ll *LingioLogger) WarningUserE(err *logicerr.Error, partnerID string, userID string) {
	m := makeUserMapFromExsisting(partnerID, userID, err.InfoMap)
	m["error_code"] = fmt.Sprintf("%v", err.HTTPStatusCode)
	m["trace"] = err.Trace
	ll.logm(err.Message, googlelog.Warning, m)
}

// WarningUserM logs a warning message
func (ll *LingioLogger) WarningUserM(message string, partnerID string, userID string, m map[string]string) {
	m = makeUserMapFromExsisting(partnerID, userID, m)
	ll.logm(message, googlelog.Warning, m)
}

// WarningM logs a warning message
func (ll *LingioLogger) WarningM(message string, m map[string]string) {
	ll.logm(message, googlelog.Warning, m)
}

// Error logs an error message
func (ll *LingioLogger) Error(message string) {
	ll.logm(message, googlelog.Error, make(map[string]string))
}

// ErrorE logs a logicerr.Error error
func (ll *LingioLogger) ErrorE(e *logicerr.Error) {
	m := e.InfoMap
	if m == nil {
		m = make(map[string]string)
	}
	m["error_code"] = fmt.Sprintf("%v", e.HTTPStatusCode)
	m["trace"] = e.Trace
	ll.logm(e.Message, googlelog.Error, m)
}

// ErrorUser logs an error message
func (ll *LingioLogger) ErrorUser(message string, partnerID string, userID string) {
	ll.logm(message, googlelog.Error, makeUserMap(partnerID, userID))
}

// ErrorUserE logs a logicerr.Error error
func (ll *LingioLogger) ErrorUserE(e *logicerr.Error, partnerID string, userID string) {
	m := makeUserMapFromExsisting(partnerID, userID, e.InfoMap)
	m["error_code"] = fmt.Sprintf("%v", e.HTTPStatusCode)
	m["trace"] = e.Trace
	ll.logm(e.Message, googlelog.Error, m)
}

// ErrorUserM logs an error message
func (ll *LingioLogger) ErrorUserM(message string, partnerID string, userID string, m map[string]string) {
	m = makeUserMapFromExsisting(partnerID, userID, m)
	ll.logm(message, googlelog.Error, m)
}

// ErrorM logs an error message
func (ll *LingioLogger) ErrorM(message string, m map[string]string) {
	ll.logm(message, googlelog.Error, m)
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

func (ll *LingioLogger) logm(message string, severity googlelog.Severity, m map[string]string) {
	if m == nil {
		m = make(map[string]string)
	}

	m["env"] = ll.env
	m["projectID"] = ll.projectID
	m["message"] = message

	if ll.sdlogger != nil {
		// Here we use the stackdriver logger

		ll.sdlogger.Log(googlelog.Entry{Payload: m, Severity: severity})

	} else {
		// Here we use the local logger
		logger, ok := ll.loggers[severity]
		if ok == false {
			fmt.Printf("Cannot log with this severity!! %v", severity)
			return
		}

		// We send 3 as the stackdepth here to that we get the right filename in the output
		_ = logger.Output(3, fmt.Sprintf("%v \n %v", message, m))
	}
}

// Flush flushes the stackdriver logger
func (ll *LingioLogger) Flush() {
	if ll.client != nil {
		err := ll.sdlogger.Flush()
		if err != nil {
			fmt.Printf("Failed flushing the stackdriver logger: %v", err)
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

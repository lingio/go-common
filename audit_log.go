package common

import (
	"context"
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// default audit logger
var auditLogger = zerolog.New(os.Stderr).With().Timestamp().Logger()

// auditLogKeyType maps context key to zerolog field.
type auditLogKeyType string

const (
	requestKey = auditLogKeyType("requestID")
	objectKey  = auditLogKeyType("objectID")
	bucketKey  = auditLogKeyType("bucketName")
	actionKey  = auditLogKeyType("action")
	authKey    = auditLogKeyType("authToken")
)

func FromEcho(e echo.Context) context.Context {
	// use the original context as parent so we support echo middleware
	ctx := e.Request().Context()

	ctx = context.WithValue(ctx, requestKey, e.Response().Header().Get(echo.HeaderXRequestID))

	// Extract JWT token. No need to verify it, that is done elsewhere.
	// We want to include the full token so we get all claims in the audit log.
	auth := e.Request().Header.Get("Authorization")
	authScheme := "Bearer"
	l := len(authScheme)
	if len(auth) <= l+1 || auth[:l] != authScheme {
		// Not all requests have an auth gate.
		return ctx
	}
	ctx = context.WithValue(ctx, authKey, auth[l+1:])

	return ctx
}

// WithObject returns a copy of the passed context with the object ID.
func WithObject(ctx context.Context, objectID string) context.Context {
	return context.WithValue(ctx, objectKey, objectID)
}

// WithBucket returns a copy of the passed context with the specified bucket name.
func WithBucket(ctx context.Context, bucketName string) context.Context {
	return context.WithValue(ctx, bucketKey, bucketName)
}

// WithAction returns a copy of the passed context with the specified action.
func WithAction(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, actionKey, action)
}

// AuthTokenFrom extracts the embedded JWT. Will panic if no token exists.
func AuthTokenFrom(ctx context.Context) string {
	return ctx.Value(authKey).(string)
}

// LogAuditEvent outputs the provided app context
func LogAuditEvent(ctx context.Context) {
	evt := auditLogger.Info()
	writeContextFieldToLogEvent(ctx, requestKey, evt)
	writeContextFieldToLogEvent(ctx, objectKey, evt)
	writeContextFieldToLogEvent(ctx, bucketKey, evt)
	writeContextFieldToLogEvent(ctx, actionKey, evt)
	// Don't print credentials.
	// writeContextFieldToLogEvent(ctx, authKey, evt)
	evt.Msg("audit log event")
}

func writeContextFieldToLogEvent(ctx context.Context, ctxKey auditLogKeyType, evt *zerolog.Event) {
	eventKey := string(ctxKey)
	val := ctx.Value(ctxKey)
	if val == nil {
		return
	} else if sval, ok := val.(string); ok {
		evt.Str(eventKey, sval)
	} else if ival, ok := val.(int); ok {
		evt.Int(eventKey, ival)
	} else {
		panic(fmt.Errorf("auditlog: cannot typecast context key %v", ctxKey))
	}
}

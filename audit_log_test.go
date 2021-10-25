package common

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestAuditLog(t *testing.T) {
	var b bytes.Buffer
	prevAuditLog := auditLog
	auditLog = auditLog.Output(&b)
	ctx := WithAction(context.TODO(), "CREATE")
	ctx = WithObject(ctx, "OBJECT")
	ctx = WithBucket(ctx, "BUCKET")
	LogAuditEvent(ctx)
	auditLog = prevAuditLog

	logmsg := make(map[string]interface{})
	if err := json.Unmarshal(b.Bytes(), &logmsg); err != nil {
		t.Fatal(err)
	}

	if action, ok := logmsg[string(actionKey)]; !ok || action != "CREATE" {
		t.Fatalf(`'action' not found or invalid: %v (%v), expected '%v'`, action, ok, "CREATE")
	}

	// TODO(Axel): Enable once we have request ID generation in place
	// if reqID, ok := logmsg[string(requestKey)]; !ok || reqID != "req-123" {
	// 	t.Fatalf(`'requestID' not found or invalid: %v (%v), expected '%v'`, reqID, ok, "req-123")
	// }

	if objID, ok := logmsg[string(objectKey)]; !ok || objID != "OBJECT" {
		t.Fatalf(`'objectID' not found or invalid: %v (%v), expected '%v'`, objID, ok, "OBJECT")
	}

	if bucketName, ok := logmsg[string(bucketKey)]; !ok || bucketName != "BUCKET" {
		t.Fatalf(`'bucketName' not found or invalid: %v (%v), expected '%v'`, bucketName, ok, "BUCKET")
	}

	// NOTE(Axel): Enable when we have a mock echo.Context with a Bearer token.
	// if authID, ok := logmsg[string(authKey)]; !ok || authID != "AUTH" {
	// 	t.Fatalf(`'authID' not found or invalid: %v (%v), expected '%v'`, authID, ok, "AUTH")
	// }

	if _, ok := logmsg["time"]; !ok {
		t.Fatalf(`timestamp not found`)
	}
}

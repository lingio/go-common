package middleware

import (
	"net/http"
	"time"

	"github.com/lingio/go-common/log"
)

type RequestLogger struct {
	ll *log.LingioLogger
}

// Apparently go ResponseWriters use a lot of "hidden" (implemented but not explicitly so)interfaces to be usable
// this creates the problem that the wrapped response writer could hide some functionality from the later stages of the request handler
// This would result in loss of functionality e.g streaming would not work because the flushing interface is not implemented
type requestLoggerResponseWriter struct {
	writer     http.ResponseWriter
	statuscode int
}

func newRequestLoggerResponseWriterResponseWriter(w http.ResponseWriter) *requestLoggerResponseWriter {
	return &requestLoggerResponseWriter{
		writer:     w,
		statuscode: 0,
	}
}

func (w *requestLoggerResponseWriter) Header() http.Header {
	return w.writer.Header()
}

func (w *requestLoggerResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *requestLoggerResponseWriter) WriteHeader(statuscode int) {
	w.writer.WriteHeader(statuscode)
	w.statuscode = statuscode
}

func CreateRequestLogger(ll *log.LingioLogger) *RequestLogger {
	reporter := &RequestLogger{}
	reporter.ll = ll
	return reporter
}

func (t *RequestLogger) RequestLogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		ctx := r.Context()
		// r.Context() can generate a new context so we make sure the request uses the context
		r.WithContext(ctx)

		w2 := newRequestLoggerResponseWriterResponseWriter(w)
		next.ServeHTTP(w2, r)

		t.ll.LogEndOfHTTPRequest(ctx, "request completed", w2.statuscode, r, time.Since(startTime))
	})
}

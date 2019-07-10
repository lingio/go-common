package requestlogger

import (
	"net/http"

	"github.com/lingio/go-common/log"
)

type RequestLogger struct {
	ll *log.LingioLogger
}

// FIXME: Apparently go ResponseWriters use a lot of "hidden" (implemented but not explicitly so)interfaces to be usable
// this creates the problem that the wrapped response writer could hide some functionaliy from the later stages of the request hander
// This would result in loss of functionality e.g streaming would not work becase the flushing interface is not implemented
type metricResponseWriter struct {
	writer     http.ResponseWriter
	statuscode int
}

func newMetricResponseWriter(w http.ResponseWriter) *metricResponseWriter {
	return &metricResponseWriter{
		writer:     w,
		statuscode: 0,
	}
}

func (w *metricResponseWriter) Header() http.Header {
	return w.writer.Header()
}

func (w *metricResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *metricResponseWriter) WriteHeader(statuscode int) {
	w.writer.WriteHeader(statuscode)
	w.statuscode = statuscode
}

func CreateRequestLogger(ll *log.LingioLogger) *RequestLogger {
	reporter := &RequestLogger{}
	reporter.ll = ll
	return reporter
}

// FIXME: We want to know the name RPC
// In this service there is only one (getCourseSchedule) but others will have more then one
// that we want to be able to separate the measurements for
func (t *RequestLogger) ReportMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// r.Context() can generate a new context so we make sure the request uses the context
		r.WithContext(ctx)

		w2 := newMetricResponseWriter(w)
		next.ServeHTTP(w2, r)

		message := w.Header().Get("message")
		if w2.statuscode >= 500 {
			t.ll.Error(ctx, message, nil)
		} else if w2.statuscode >= 400 {
			t.ll.Warning(ctx, message, nil)
		} else {
			t.ll.Info(ctx, message, nil)
		}
	})
}

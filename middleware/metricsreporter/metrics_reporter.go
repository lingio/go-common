package metricsreporter

import (
	"net/http"
	"strconv"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	// NOTE: Is it important that these names are valid or the program will panic on startup
	KeyStatus    tag.Key = tag.MustNewKey("status")
	KeyError     tag.Key = tag.MustNewKey("error")
	KeyErrorCode tag.Key = tag.MustNewKey("error_code")
	KeyMethod    tag.Key = tag.MustNewKey("method")
)

type MetricsReporter struct {
	// MLatencyMs The latency in milliseconds
	MLatencyMs *stats.Float64Measure
	// MErrorsCount The number of errors generated
	MErrorsCount *stats.Int64Measure

	LatencyView    *view.View
	ErrorCountView *view.View
}

// FIXME: Apparently go ResponseWriters use a lot of "hidden" (implemented but not explicitly so)interfaces to be usable
// this creates the problem that the wrapped response writer could hide some functionaliy from the later stages of the request hander
// This would result in loss of functionality e.g streaming would not work becase the flushing interface is not implemented
// There is a package that aims to fix this: https://github.com/felixge/httpsnoop
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

func CreateMetricsReporter() *MetricsReporter {
	reporter := &MetricsReporter{}

	reporter.MLatencyMs = stats.Float64("get_course_schedule/latency", "The latency in milliseconds per request", stats.UnitMilliseconds)
	reporter.MErrorsCount = stats.Int64("get_course_schedule/errors", "The number of errors generated", "{tot}")

	reporter.LatencyView = &view.View{
		Name:        "get_course_schedule/latency",
		Measure:     reporter.MLatencyMs,
		Description: "The distribution of the latencies",

		// Latency in buckets:
		// [>=0ms, >=25ms, >=50ms, >=75ms, >=100ms, >=200ms, >=400ms, >=600ms, >=800ms, >=1s, >=2s, >=4s, >=6s]
		Aggregation: view.Distribution(25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 6000),
		TagKeys:     []tag.Key{KeyMethod, KeyStatus}}

	reporter.ErrorCountView = &view.View{
		Name:        "get_course_schedule/errors",
		Measure:     reporter.MErrorsCount,
		Description: "The number of errors encountered",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyMethod, KeyErrorCode}}

	view.Register(reporter.LatencyView, reporter.ErrorCountView)

	return reporter
}

// FIXME: We want to know the name RPC
// In this service there is only one (getCourseSchedule) but others will have more then one
// that we want to be able to separate the measurements for
func (t *MetricsReporter) ReportMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// r.Context() can generate a new context so we make sure the request uses the context
		r.WithContext(ctx)
		startTime := time.Now()

		w2 := newMetricResponseWriter(w)
		next.ServeHTTP(w2, r)

		if w2.statuscode >= 400 {
			ctx, _ = tag.New(ctx, tag.Upsert(KeyStatus, "ERROR"))
			stats.Record(ctx, t.MErrorsCount.M(1))
		} else {
			ctx, _ = tag.New(ctx, tag.Upsert(KeyStatus, "OK"))
		}

		// FIMXE: We want to tag the metric with the appropriate errormessage!
		// But we can't really as the context is not preserved

		ctx, _ = tag.New(ctx, tag.Upsert(KeyErrorCode, strconv.Itoa(w2.statuscode)))
		stats.Record(ctx, t.MLatencyMs.M(float64(time.Since(startTime).Nanoseconds())/1e6))
	})
}

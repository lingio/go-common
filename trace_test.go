package common

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// The Authorization header must reach Tempo, both from the config field and
// from the TEMPO_AUTHORIZATION env var.
func TestInitHttpTraceSendsAuthorization(t *testing.T) {
	for name, set := range map[string]func(*MonitorConfig){
		"field": func(m *MonitorConfig) { m.TempoAuthorization = "Basic field" },
		"env":   func(m *MonitorConfig) { t.Setenv("TEMPO_AUTHORIZATION", "Basic env") },
	} {
		t.Run(name, func(t *testing.T) {
			got := make(chan string, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got <- r.Header.Get("Authorization")
			}))
			defer srv.Close()

			mon := MonitorConfig{TempoHost: strings.TrimPrefix(srv.URL, "http://")}
			set(&mon)
			if err := InitHttpTrace("test", EnvDevelop, mon); err != nil {
				t.Fatal(err)
			}
			_, span := otel.Tracer("test").Start(context.Background(), "op")
			span.End()
			if err := otel.GetTracerProvider().(*sdktrace.TracerProvider).ForceFlush(context.Background()); err != nil {
				t.Fatal(err)
			}
			if h := <-got; h != "Basic "+name {
				t.Fatalf("Authorization = %q", h)
			}
		})
	}
}

package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestMetricsLabels(t *testing.T) {
	e := echo.New()
	e.GET("/users/:id", func(c echo.Context) error { return nil })

	label := func(host, path string) (string, string) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = host
		c := e.NewContext(req, httptest.NewRecorder())
		e.Router().Find(http.MethodGet, path, c)
		return metricsURLLabel(c), metricsHostLabel(c)
	}

	tests := []struct{ host, path, url, hostLabel string }{
		{"token-service.api-eu1.lingio.com", "/users/42", "/users/:id", "token-service.api-eu1.lingio.com"},
		{"Token-Service.API-EU1.lingio.com:443", "/wp-login.php", "unmatched", "token-service.api-eu1.lingio.com"},
		{"enrollment-service.default.svc.cluster.local:8080", "/users/1", "/users/:id", "enrollment-service.default.svc.cluster.local"},
		{"10.52.4.2:4711", "/.env", "unmatched", "other"},
		{"evil.example.com", "/users/1", "/users/:id", "other"},
		{"", "/users/1", "/users/:id", "other"},
	}
	for _, tt := range tests {
		url, host := label(tt.host, tt.path)
		if url != tt.url || host != tt.hostLabel {
			t.Errorf("host %q path %q: got url=%q host=%q, want url=%q host=%q", tt.host, tt.path, url, host, tt.url, tt.hostLabel)
		}
	}
}

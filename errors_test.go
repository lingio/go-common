package common

import (
	"bufio"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestEchoErrorHandler(t *testing.T) {
	var (
		e        = echo.New()
		notFound = Errorf(ErrObjectNotFound, "xyz")
	)

	var variants = []struct {
		err  error
		code int
	}{
		{ErrObjectNotFound, http.StatusInternalServerError},                // raw return is an implementation error
		{notFound, http.StatusNotFound},                                    // wrapped with msg is good and returns 404
		{Errorf(notFound).Str("key", "val"), http.StatusNotFound},          // try with attribute
		{Errorf(notFound, "wrapped w/o code"), http.StatusNotFound},        // wrapping an err should "bubble" its code
		{Errorf(notFound, "wrapped w/code", http.StatusOK), http.StatusOK}, // unless code is specified
		{Errorf(Errorf(Errorf(notFound), http.StatusOK)), http.StatusOK},   // should work even in more complex error trees
	}

	for _, tc := range variants {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		c := e.NewContext(req, rec)

		var echoError *echo.HTTPError
		if errors.As(tc.err, &echoError) {
			e.DefaultHTTPErrorHandler(echoError, c)
		} else {
			e.DefaultHTTPErrorHandler(tc.err, c)
		}

		if rec.Code != tc.code {
			t.Errorf("expected http code %v but got %v", tc.code, rec.Code)
		}
		if !errors.Is(tc.err, ErrObjectNotFound) { // check that variants satisfy this common pattern
			t.Errorf("err did not match source ErrObjectNotFound: %v", tc.err)
		}
	}
}

func TestErrorTrace(t *testing.T) {
	var variants = []struct {
		err   error
		trace string
	}{
		{ErrObjectNotFound, `> error: object not found`},
		{Errorf(Errorf(ErrObjectNotFound, "xyz"), "abc").Str("key", "val"), strings.TrimSpace(`
			> location (404): "abc"
			| [map] key:val
			\ location (404): "xyz"
			  \ error: object not found`),
		},
	}

	var location = regexp.MustCompile(`[\w_\.]+\:\d+`)

	for _, tc := range variants {
		trace := FullErrorTrace(tc.err)

		actual := bufio.NewScanner(strings.NewReader(trace))
		expect := bufio.NewScanner(strings.NewReader(tc.trace))
		for actual.Scan() && expect.Scan() {
			var (
				actual = strings.TrimSpace(actual.Text())
				expect = strings.TrimSpace(expect.Text())
			)

			actual = location.ReplaceAllString(actual, "location")

			if actual != expect {
				t.Errorf("line %q != %q", expect, actual)
			}
		}
	}
}

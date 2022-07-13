package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

/* 	Fetch from remote service
Usage example:
	b, lerr := HttpGet(fmt.Sprintf("%s/courses/%s", host, courseID), token)
	if lerr != nil {
		return nil, lerr
	}
	var cr CourseResponse
	err := json.Unmarshal(b, &cr)
*/

var commonClient = http.Client{
	Transport: otelhttp.NewTransport(&http.Transport{
		MaxIdleConnsPerHost:   10,
		MaxIdleConns:          100,
		IdleConnTimeout:       time.Second * 60,
		ResponseHeaderTimeout: time.Second * 30,
		TLSHandshakeTimeout:   time.Second * 30,
	}, otelhttp.WithTracerProvider(nil)),
}

type RemoteError struct {
	Message string `json:"message"`
}

func HttpGet(ctx context.Context, url string, bearerToken string) (_ []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, NewErrorE(http.StatusInternalServerError, err).Msg("failed to create request")
	}
	setBearerToken(req, bearerToken)

	shouldRetry := func(code, attempt int) bool {
		return isHttpStatusRetryable(code) && attempt < 3
	}

	return executeReqWithRetry(req, shouldRetry, exponentialBackoff)
}

func HttpPost(ctx context.Context, url string, body interface{}, bearerToken string) ([]byte, error) {
	bodyBuffer, lerr := createBodyBuffer(body)
	if lerr != nil {
		return []byte{}, nil
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyBuffer)
	if err != nil {
		return nil, NewErrorE(http.StatusInternalServerError, err).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	setBearerToken(req, bearerToken)
	return executeReq(req)
}

func HttpPut(ctx context.Context, url string, body interface{}, bearerToken string) ([]byte, error) {
	bodyBuffer, lerr := createBodyBuffer(body)
	if lerr != nil {
		return []byte{}, nil
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bodyBuffer)
	if err != nil {
		return nil, NewErrorE(http.StatusInternalServerError, err).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	setBearerToken(req, bearerToken)
	return executeReq(req)
}

func HttpDelete(ctx context.Context, url string, bearerToken string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, NewErrorE(http.StatusInternalServerError, err).Msg("failed to create request")
	}
	setBearerToken(req, bearerToken)
	return executeReq(req)
}

func HttpGetWithApiKey(ctx context.Context, url string, apiKey string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, NewErrorE(http.StatusInternalServerError, err).Msg("failed to create request")
	}
	setApiKey(req, apiKey)
	return executeReq(req)
}

func HttpPostWithApiKey(ctx context.Context, url string, body interface{}, apiKey string) ([]byte, error) {
	bodyBuffer, lerr := createBodyBuffer(body)
	if lerr != nil {
		return []byte{}, nil
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyBuffer)
	if err != nil {
		return nil, NewErrorE(http.StatusInternalServerError, err).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	setApiKey(req, apiKey)
	return executeReq(req)
}

func HttpPutWithApiKey(ctx context.Context, url string, body interface{}, apiKey string) ([]byte, error) {
	bodyBuffer, lerr := createBodyBuffer(body)
	if lerr != nil {
		return []byte{}, nil
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bodyBuffer)
	if err != nil {
		return nil, NewErrorE(http.StatusInternalServerError, err).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	setApiKey(req, apiKey)
	return executeReq(req)
}

func executeReq(req *http.Request) ([]byte, *Error) {
	resp, err := commonClient.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		lerr := NewErrorE(http.StatusBadGateway, err).
			Str("host", req.URL.Host).
			Str("url", req.URL.Path).
			Msg("error calling remote service")
		if resp != nil {
			lerr.HttpStatusCode = resp.StatusCode
		}
		return nil, lerr
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, NewErrorE(http.StatusInternalServerError, err).Msg("failed reading the response")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var x RemoteError
		if err := json.Unmarshal(data, &x); err != nil {
			return nil, NewError(resp.StatusCode).
				Str("host", req.URL.Host).
				Str("url", req.URL.Path).
				Int("remoteStatusCode", resp.StatusCode).
				Msg("remote error without message")
		}
		return nil, NewError(resp.StatusCode).
			Str("url", req.URL.Path).
			Int("remoteStatusCode", resp.StatusCode).
			Str("remoteError", x.Message).
			Msg("remote error")
	}
	return data, nil
}

// ShouldRetry returns whether a request can be retried:
//    code    = http status code
//    attempt = 0-based retry counter
type ShouldRetry func(code, attempt int) bool

// Backoff returns the time to wait for request retry attempt.
type Backoff func(attempt int) time.Duration

func executeReqWithRetry(req *http.Request, shouldRetry ShouldRetry, backoff Backoff) ([]byte, error) {
	var attempt int
	for {
		data, err := executeReq(req)
		if err == nil {
			return data, nil
		}

		if shouldRetry(err.HttpStatusCode, attempt) {
			time.Sleep(backoff(attempt))
			attempt++
			continue
		}

		return nil, err
	}
}

func setBearerToken(req *http.Request, bearerToken string) {
	bearerHeader := fmt.Sprintf("Bearer %s", bearerToken)
	if bearerToken != "" {
		req.Header.Add("Authorization", bearerHeader)
	}
}

func setApiKey(req *http.Request, key string) {
	req.Header.Add("x-api-key", key)
}

func createBodyBuffer(body interface{}) (*bytes.Buffer, error) {
	var bodyBuffer *bytes.Buffer
	if body != nil {
		jsonValue, err := json.Marshal(body)
		if err != nil {
			return nil, NewErrorE(http.StatusInternalServerError, err).Msg("failed json marshal")
		}
		bodyBuffer = bytes.NewBuffer(jsonValue)
	}
	return bodyBuffer, nil
}

// exponentialBackoff returns a backoff function:
//   y = 2^attempt (seconds)
// where
//	attempt = min(attempt, 6)
// which implies that max wait is 64s
func exponentialBackoff(attempt int) time.Duration {
	if attempt > 6 {
		attempt = 6
	}
	n := math.Pow(2, float64(attempt))
	t := time.Duration(int64(time.Second) * int64(n))
	return t
}

// isHttpStatusRetryable
func isHttpStatusRetryable(statusCode int) bool {
	// Enable these carefully! Might have unexpected effects on service behavior.
	switch statusCode {
	// case http.StatusRequestTimeout: // Perhaps too loaded
	// case http.StatusTooEarly: // Resource not yet ready
	case http.StatusTooManyRequests: // Rate limited
	case http.StatusBadGateway: // Cluster config / dns
	// case http.StatusServiceUnavailable: // Cluster config / dns
	// case http.StatusGatewayTimeout: // Cluster config / dns
	default:
		return false
	}

	return true
}

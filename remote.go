package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

type RemoteError struct {
	Message string `json:"message"`
}

func HttpGet(url string, bearerToken string) ([]byte, *Error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, NewError(http.StatusInternalServerError).Msg("failed to create request")
	}
	setBearerToken(req, bearerToken)
	return executeReq(req)
}

func HttpPost(url string, body interface{}, bearerToken string) ([]byte, *Error) {
	bodyBuffer, lerr := createBodyBuffer(body)
	if lerr != nil {
		return []byte{}, nil
	}
	req, err := http.NewRequest("POST", url, bodyBuffer)
	if err != nil {
		return nil, NewError(http.StatusInternalServerError).Str("err", err.Error()).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	setBearerToken(req, bearerToken)
	return executeReq(req)
}

func HttpPut(url string, body interface{}, bearerToken string) ([]byte, *Error) {
	bodyBuffer, lerr := createBodyBuffer(body)
	if lerr != nil {
		return []byte{}, nil
	}
	req, err := http.NewRequest("PUT", url, bodyBuffer)
	if err != nil {
		return nil, NewError(http.StatusInternalServerError).Str("err", err.Error()).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	setBearerToken(req, bearerToken)
	return executeReq(req)
}

func HttpDelete(url string, bearerToken string) ([]byte, *Error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, NewError(http.StatusInternalServerError).Msg("failed to create request")
	}
	setBearerToken(req, bearerToken)
	return executeReq(req)
}

func HttpGetWithApiKey(url string, apiKey string) ([]byte, *Error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, NewError(http.StatusInternalServerError).Msg("failed to create request")
	}
	setApiKey(req, apiKey)
	return executeReq(req)
}

func HttpPostWithApiKey(url string, body interface{}, apiKey string) ([]byte, *Error) {
	bodyBuffer, lerr := createBodyBuffer(body)
	if lerr != nil {
		return []byte{}, nil
	}
	req, err := http.NewRequest("POST", url, bodyBuffer)
	if err != nil {
		return nil, NewError(http.StatusInternalServerError).Str("err", err.Error()).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	setApiKey(req, apiKey)
	return executeReq(req)
}

func HttpPutWithApiKey(url string, body interface{}, apiKey string) ([]byte, *Error) {
	bodyBuffer, lerr := createBodyBuffer(body)
	if lerr != nil {
		return []byte{}, nil
	}
	req, err := http.NewRequest("PUT", url, bodyBuffer)
	if err != nil {
		return nil, NewError(http.StatusInternalServerError).Str("err", err.Error()).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	setApiKey(req, apiKey)
	return executeReq(req)
}

func executeReq(req *http.Request) ([]byte, *Error) {
	resp, err := http.DefaultClient.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			return nil, NewError(resp.StatusCode).Str("err", err.Error()).Str("host", req.URL.Host).Str("url", req.URL.Path).Msg("error calling remote service")
		}
		return nil, NewError(http.StatusBadGateway).Str("err", err.Error()).Str("host", req.URL.Host).Str("url", req.URL.Path).Msg("error calling remote service")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, NewError(http.StatusInternalServerError).Str("err", err.Error()).Msg("failed reading the response")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var x RemoteError
		err := json.Unmarshal(data, &x)
		if err != nil {
			return nil, NewError(http.StatusBadGateway).Str("host", req.URL.Host).Str("url", req.URL.Path).
				Int("remoteStatusCode", resp.StatusCode).Msg("remote error without message")
		}
		return nil, NewError(resp.StatusCode).Str("url", req.URL.Path).
			Int("remoteStatusCode", resp.StatusCode).Str("remoteError", x.Message).Msg("remote error")
	}
	return data, nil
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

func createBodyBuffer(body interface{}) (*bytes.Buffer, *Error) {
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

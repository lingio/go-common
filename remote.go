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

func HttpGet(url string, bearerToken string) ([]byte, *Error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, NewError(http.StatusInternalServerError).Msg("failed to create request")
	}
	setBearerToken(req, bearerToken)
	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, NewErrorE(http.StatusBadGateway, err).Str("url", url).Msg("error calling remote service")
	} else if resp.StatusCode != 200 {
		return nil, NewError(http.StatusBadGateway).Str("url", url).Int("remoteStatusCode", resp.StatusCode).Msg("status code error")
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, NewErrorE(http.StatusInternalServerError, err).Msg("failed reading response")
	}
	return b, nil
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, NewError(http.StatusBadGateway).Str("err", err.Error()).Str("url", url).Msg("error calling remote service")
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, NewError(http.StatusNotFound).Str("url", url).Int("remoteStatusCode", resp.StatusCode).Msg("not found")
	} else if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, NewError(http.StatusBadGateway).Str("url", url).Int("remoteStatusCode", resp.StatusCode).Msg("status code error")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, NewError(http.StatusInternalServerError).Str("err", err.Error()).Msg("failed reading the response")
	}
	return data, nil
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, NewErrorE(http.StatusBadGateway, err).Str("url", url).Msg("error calling remote service")
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, NewError(http.StatusNotFound).Str("url", url).Int("remoteStatusCode", resp.StatusCode).Msg("not found")
	} else if resp.StatusCode != 200 {
		return nil, NewError(http.StatusBadGateway).Str("url", url).Int("remoteStatusCode", resp.StatusCode).Msg("status code error")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, NewError(http.StatusInternalServerError).Str("err", err.Error()).Msg("failed reading the response")
	}
	return data, nil
}

func setBearerToken(req *http.Request, bearerToken string) {
	bearerHeader := fmt.Sprintf("Bearer %s", bearerToken)
	if bearerToken != "" {
		req.Header.Add("Authorization", bearerHeader)
	}
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

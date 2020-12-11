package common

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

/* 	Fetch from remote service
Usage example:
	b, lerr := get(fmt.Sprintf("%s/courses/%s", host, courseID))
	if lerr != nil {
		return nil, lerr
	}
	var cr CourseResponse
	err := json.Unmarshal(b, &cr)
*/

func HttpGet(url string) ([]byte, *Error) {
	resp, err := http.Get(url)
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

func HttpPost(url string, body interface{}) ([]byte, *Error) {
	jsonValue, err := json.Marshal(body)
	if err != nil {
		return []byte{}, NewErrorE(http.StatusInternalServerError, err).Msg("failed json marshal")
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, NewErrorE(http.StatusBadGateway, err).Str("url", url).Msg("error calling remote service")
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, NewError(http.StatusNotFound).Str("url", url).Int("remoteStatusCode", resp.StatusCode).Msg("not found")
	} else if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, NewError(http.StatusBadGateway).Str("url", url).Int("remoteStatusCode", resp.StatusCode).Msg("status code error")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, NewErrorE(http.StatusInternalServerError, err).Msg("failed reading the response")
	}
	return data, nil
}

func HttpPostNoBody(url string) ([]byte, *Error) {
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return nil, NewErrorE(http.StatusBadGateway, err).Str("url", url).Msg("error calling remote service")
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, NewError(http.StatusNotFound).Str("url", url).Int("remoteStatusCode", resp.StatusCode).Msg("not found")
	} else if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, NewError(http.StatusBadGateway).Str("url", url).Int("remoteStatusCode", resp.StatusCode).Msg("status code error")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, NewErrorE(http.StatusInternalServerError, err).Msg("failed reading the response")
	}
	return data, nil
}

func HttpPut(url string, body interface{}) ([]byte, *Error) {
	jsonValue, err := json.Marshal(body)
	if err != nil {
		return []byte{}, NewErrorE(http.StatusInternalServerError, err).Msg("failed json marshal")
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return []byte{}, NewErrorE(http.StatusInternalServerError, err).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
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
		return []byte{}, NewErrorE(http.StatusInternalServerError, err).Msg("failed reading the response")
	}
	return data, nil
}

func HttpPutNoBody(url string) ([]byte, *Error) {
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return []byte{}, NewErrorE(http.StatusInternalServerError, err).Msg("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
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
		return []byte{}, NewErrorE(http.StatusInternalServerError, err).Msg("failed reading the response")
	}
	return data, nil
}

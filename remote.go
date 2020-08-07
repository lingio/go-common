package common

import (
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
	defer resp.Body.Close()
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

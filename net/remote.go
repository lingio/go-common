package net

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lingio/go-common/logicerr"
	"go.opencensus.io/exporter/stackdriver/propagation"
	"go.opencensus.io/plugin/ochttp"
)

func CallRemoteService(ctx context.Context, url string) (*http.Response, error) {
	client := &http.Client{
		Transport: &ochttp.Transport{
			Propagation: &propagation.HTTPFormat{},
		},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &logicerr.Error{ HttpStatusCode: resp.StatusCode, Message: fmt.Sprintf("Failed call to %s", url)}
	}

	return resp, nil
}

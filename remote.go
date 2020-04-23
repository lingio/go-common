package common

import (
	"context"
	"net/http"

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
	if err != nil {
		resp.Body.Close()
		return nil, err
	}

	return resp, nil
}

package common

import (
	"fmt"
	"net"
	"net/http"

	zl "github.com/rs/zerolog/log"
)

// StatusProbe describes a simple interface for querying service status.
type StatusProbe interface {
	Started() bool
	Ready() bool
	Live() bool
}

func StatusProbeServer(port int, probes ...StatusProbe) *http.Server {
	started := func() bool {
		for _, probe := range probes {
			if !probe.Started() {
				return false
			}
		}
		return true
	}

	ready := func() bool {
		for _, probe := range probes {
			if !probe.Ready() {
				return false
			}
		}
		return true
	}

	live := func() bool {
		for _, probe := range probes {
			if !probe.Live() {
				return false
			}
		}
		return true
	}

	handler := func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/startup" && started() {
			res.WriteHeader(http.StatusOK)
		} else if req.URL.Path == "/ready" && ready() {
			res.WriteHeader(http.StatusOK)
		} else if req.URL.Path == "/live" && live() {
			res.WriteHeader(http.StatusOK)
		} else {
			res.WriteHeader(http.StatusInternalServerError)
		}

		res.Write([]byte("HELO"))
	}

	srv := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", fmt.Sprint(port)),
		Handler: http.HandlerFunc(handler),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zl.Fatal().Str("error", err.Error()).Msg("fatal error serving status probes")
		}
	}()

	return srv
}

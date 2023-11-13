package common

import (
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

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

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/debug/pprof", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", fmt.Sprint(port)),
		Handler: mux,
	}

	go func() {
		zl.Info().Str("addr", srv.Addr).Msg("starting statusprobe server")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zl.Error().Str("error", err.Error()).Msg("error serving status probes")
		}
	}()

	return srv
}

func ping(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("PONG"))
}

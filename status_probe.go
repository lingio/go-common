package common

import (
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

	zl "github.com/rs/zerolog/log"
)

// StatusProbe describes a simple interface for querying service status.
// Handlers can be called at any time from multiple goroutines.
type StatusProbe interface {
	Started() bool
	Ready() bool
	Live() bool
}

// StatusProbeServer starts a http server on 0.0.0.0:port with the specified
// probes. The returned server may be shutdown at any given time by the caller.
//
//	GET /startup
//	GET /ready
//	GET /live
//
// The endpoints will only return code 200 if all probes are ok for the given handler.
// Otherwise, they return status code 500.
//
// The existance check endpoint will always return 200 "PONG":
//
//	GET /ping
//
// Go pprof:
//
//	GET /pprof
//
// Run on k8s instance:
//
//	kubectl port-forward <pod> 9999:9999
//
// Examples:
//
//	go tool pprof -http localhost:3333 http://localhost:9999/pprof/profile
//	go tool pprof -http localhost:3333 http://localhost:9999/pprof/heap
//	go tool pprof -http localhost:3333 http://localhost:9999/pprof/trace
func StatusProbeServer(port int, probes ...StatusProbe) *http.Server {
	newprobehandler := func(checkfn func(StatusProbe) bool) func(http.ResponseWriter, *http.Request) {
		return func(res http.ResponseWriter, req *http.Request) {
			var failed int
			for _, probe := range probes {
				if !checkfn(probe) {
					failed++
				}
			}
			if failed == 0 {
				res.WriteHeader(http.StatusOK)
				res.Write([]byte("ok"))
			} else {
				res.WriteHeader(http.StatusInternalServerError)
				res.Write([]byte(fmt.Sprintf("%d probes failed", failed)))
			}
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)

	mux.HandleFunc("/startup", newprobehandler(probeStarted))
	mux.HandleFunc("/ready", newprobehandler(probeReady))
	mux.HandleFunc("/live", newprobehandler(probeLive))

	mux.HandleFunc("/debug/pprof", pprof.Index)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
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
			zl.Fatal().Str("error", err.Error()).Msg("error serving status probes")
		}
	}()

	return srv
}

func probeStarted(probe StatusProbe) bool { return probe.Started() }
func probeReady(probe StatusProbe) bool   { return probe.Ready() }
func probeLive(probe StatusProbe) bool    { return probe.Live() }

func ping(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("PONG"))
}

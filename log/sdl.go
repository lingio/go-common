package log

import (
	"context"
	"fmt"
	golog "log"
	"os"

	googlelog "cloud.google.com/go/logging"
)

type lingioSDL struct {
	sdl    *googlelog.Logger
	m      map[string]string
	stdl   *golog.Logger
	client *googlelog.Client
}

func NewLingioSDL(projectID string, serviceName string, params map[string]string) *lingioSDL {

	// Set up Google Cloud Stackdriver logger
	ctx := context.Background()
	client, err := googlelog.NewClient(ctx, projectID)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %s", err)
		os.Exit(1)
	}
	sdl := client.Logger(serviceName)

	stdl := sdl.StandardLogger(googlelog.Info)

	// Create Lingio Logger that wraps Stackdriver logger
	return &lingioSDL{sdl: sdl, m: params, stdl: stdl, client: client}
}

func (ll *lingioSDL) Debug(msg string) {
	ll.log(msg, googlelog.Debug)
}

func (ll *lingioSDL) Debug1(msg string, k1 string, v1 string) {
	ll.m[k1] = v1
	ll.log(msg, googlelog.Debug)
}

func (ll *lingioSDL) DebugParams(msg string, m map[string]string) {
	for k, v := range m {
		ll.m[k] = v
	}
	ll.log(msg, googlelog.Debug)
}


func (ll *lingioSDL) Info(msg string) {
	ll.log(msg, googlelog.Info)
}

func (ll *lingioSDL) Info1(msg string, k1 string, v1 string) {
	ll.m[k1] = v1
	ll.log(msg, googlelog.Info)
}

func (ll *lingioSDL) InfoParams(msg string, m map[string]string) {
	for k, v := range m {
		ll.m[k] = v
	}
	ll.log(msg, googlelog.Info)
}


func (ll *lingioSDL) Warn(msg string) {
	ll.log(msg, googlelog.Warning)
}

func (ll *lingioSDL) Warn1(msg string, k1 string, v1 string) {
	ll.m[k1] = v1
	ll.log(msg, googlelog.Warning)
}

func (ll *lingioSDL) WarnParams(msg string, m map[string]string) {
	for k, v := range m {
		ll.m[k] = v
	}
	ll.log(msg, googlelog.Warning)
}


func (ll *lingioSDL) Err(msg string, e error) {
	ll.m["message"] = msg
	ll.m["error"] = e.Error()
	ll.sdl.Log(googlelog.Entry{Payload: ll.m, Severity: googlelog.Error})
}


func (ll *lingioSDL) log(msg string, severity googlelog.Severity) {
	ll.m["message"] = msg
	ll.sdl.Log(googlelog.Entry{Payload: ll.m, Severity: severity})
}

func (ll *lingioSDL) StandardLogger() *golog.Logger {
	return ll.stdl
}

func (ll *lingioSDL) Shutdown() {
	if err := ll.client.Close(); err != nil {
		fmt.Printf("Failed to close client: %v", err)
	}
}

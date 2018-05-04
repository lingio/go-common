package log

import "cloud.google.com/go/logging"

type lingioLogger struct {
	sdl *logging.Logger
	m   map[string]string
}

func NewLingioLogger(sdl *logging.Logger, m map[string]string) *lingioLogger {
	return &lingioLogger{sdl: sdl, m: m}
}

func (ll *lingioLogger) LogDebug(msg string) {
	ll.log(msg, logging.Debug)
}

func (ll *lingioLogger) LogInfo(msg string) {
	ll.log(msg, logging.Info)
}

func (ll *lingioLogger) LogWarn(msg string) {
	ll.log(msg, logging.Warning)
}

func (ll *lingioLogger) LogWarn1(msg string, k1 string, v1 string) {
	ll.m[k1] = v1
	ll.log(msg, logging.Warning)
}

func (ll *lingioLogger) LogErr(msg string, e error) {
	ll.m["message"] = msg
	ll.m["error"] = e.Error()
	ll.sdl.Log(logging.Entry{Payload: ll.m, Severity: logging.Error})
}

func (ll *lingioLogger) log(msg string, severity logging.Severity) {
	ll.m["message"] = msg
	ll.sdl.Log(logging.Entry{Payload: ll.m, Severity: severity})
}

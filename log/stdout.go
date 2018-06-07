package log

import (
	"fmt"
)

type stdout struct {
}

func NewStdout() *stdout {
	return &stdout{}
}

func (sl *stdout) Debug(msg string) {
	sl.log("DEBUG", msg)
}

func (sl *stdout) Debug1(msg string, k1 string, v1 string) {
	params := map[string]string{k1: v1}
	sl.logWithParams("DEBUG", msg, params)
}

func (sl *stdout) DebugParams(msg string, params map[string]string) {
	sl.logWithParams("DEBUG", msg, params)
}

func (sl *stdout) Info(msg string) {
	sl.log("INFO", msg)
}

func (sl *stdout) Info1(msg string, k1 string, v1 string) {
	params := map[string]string{k1: v1}
	sl.logWithParams("INFO", msg, params)
}

func (sl *stdout) InfoParams(msg string, params map[string]string) {
	sl.logWithParams("INFO", msg, params)
}

func (sl *stdout) Warn(msg string) {
	sl.log("WARNING", msg)
}

func (sl *stdout) Warn1(msg string, k1 string, v1 string) {
	params := map[string]string{k1: v1}
	sl.logWithParams("WARNING", msg, params)
}

func (sl *stdout) WarnParams(msg string, m map[string]string) {
	sl.logWithParams("WARNING", msg, m)
}

func (sl *stdout) Err(msg string, e error) {
	sl.log("ERROR", e.Error())
}

func (sl *stdout) logWithParams(severity string, msg string, params map[string]string) {
	fmt.Printf("%s %s. %s", severity, msg, params)
}

func (sl *stdout) log(severity string, msg string) {
	fmt.Printf("%s %s", severity, msg)
}

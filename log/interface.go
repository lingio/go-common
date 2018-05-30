package log


type LingioLog interface {
	Debug(msg string)
	Debug1(msg string, k1 string, v1 string)
	DebugParams(msg string, m map[string]string)

	Info(msg string)
	Info1(msg string, k1 string, v1 string)
	InfoParams(msg string, m map[string]string)

	Warn(msg string)
	Warn1(msg string, k1 string, v1 string)
	WarnParams(msg string, m map[string]string)

	Err(msg string, e error)
}

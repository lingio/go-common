package common

import (
	"time"

	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
)

func Bool(b *bool) bool {
	if b != nil && *b {
		return true
	}
	return false
}

func Int(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func IntP(i int) *int {
	return &i
}

func Str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func Time(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func Date(d *openapi_types.Date) openapi_types.Date {
	if d == nil {
		return openapi_types.Date{}
	}
	return *d
}

func StrP(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func TimeP(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func DateP(d openapi_types.Date) *openapi_types.Date {
	if d.IsZero() {
		return nil
	}
	return &d
}

func DatePFromTime(t time.Time) *openapi_types.Date {
	if t.IsZero() {
		return nil
	}
	return &openapi_types.Date{Time: t}
}

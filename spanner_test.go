package common

import (
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
)

func TestEncodeSpannerStructFields(t *testing.T) {
	type StrType string

	testcases := []struct {
		name   string
		source any
		target any
		wanted any
	}{
		{
			name:   "string value (native)",
			source: &struct{ S string }{"u1"},
			target: &struct{ S string }{},
			wanted: &struct{ S string }{"u1"},
		},
		{
			name:   "int64 value (native)",
			source: &struct{ X int64 }{10},
			target: &struct{ X int64 }{},
			wanted: &struct{ X int64 }{10},
		},
		{
			name:   "bool value (native)",
			source: &struct{ B bool }{true},
			target: &struct{ B bool }{},
			wanted: &struct{ B bool }{true},
		},
		{
			name:   "int32 value to int64 value (converted)",
			source: &struct{ X int32 }{10},
			target: &struct{ X int64 }{},
			wanted: &struct{ X int64 }{10},
		},
		{
			name:   "string pointer to NullString",
			source: &struct{ S *string }{StrP("u2")},
			target: &struct{ S spanner.NullString }{},
			wanted: &struct{ S spanner.NullString }{spanner.NullString{"u2", true}},
		},
		{
			name:   "int pointer to NullInt64",
			source: &struct{ X *int }{IntP(10)},
			target: &struct{ X spanner.NullInt64 }{},
			wanted: &struct{ X spanner.NullInt64 }{spanner.NullInt64{10, true}},
		},
		{
			name:   "null int pointer to NullInt64",
			source: &struct{ X *int }{},
			target: &struct{ X spanner.NullInt64 }{},
			wanted: &struct{ X spanner.NullInt64 }{},
		},
		{
			name:   "null int64 pointer to NullInt64",
			source: &struct{ X *int64 }{},
			target: &struct{ X spanner.NullInt64 }{},
			wanted: &struct{ X spanner.NullInt64 }{},
		},
		{
			name:   "null string pointer to NullString",
			source: &struct{ S *string }{nil},
			target: &struct{ S spanner.NullString }{},
			wanted: &struct{ S spanner.NullString }{},
		},
		{
			name:   "openapi_types.Date to time.Time",
			source: &struct{ D openapi_types.Date }{D: openapi_types.Date{time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)}},
			target: &struct{ D time.Time }{},
			wanted: &struct{ D time.Time }{D: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)},
		},
		{
			name: "complex struct",
			source: &struct {
				S string
				X int
				M map[string]interface{}
			}{"X", 100, map[string]interface{}{
				"SECRET": "42",
				"COMPLEX": map[string]interface{}{
					"SUBKEY": 99,
				},
			}},
			target: &struct {
				S string
				X int64
				M string `spannerType:"jsonstring"`
			}{},
			wanted: &struct {
				S string
				X int64
				M string `spannerType:"jsonstring"`
			}{"X", 100, "{\"COMPLEX\":{\"SUBKEY\":99},\"SECRET\":\"42\"}"},
		},
		{
			name:   "locally defined string type",
			source: &struct{ S StrType }{StrType("hej")},
			target: &struct{ S string }{},
			wanted: &struct{ S string }{"hej"},
		},
		{
			name:   "ptr to locally defined string type to nullstring",
			source: &struct{ S *StrType }{(*StrType)(StrP("hej"))},
			target: &struct{ S spanner.NullString }{},
			wanted: &struct{ S spanner.NullString }{spanner.NullString{"hej", true}},
		},
		{
			name:   "null locally defined string type to nullstring",
			source: &struct{ S *StrType }{nil},
			target: &struct{ S spanner.NullString }{},
			wanted: &struct{ S spanner.NullString }{spanner.NullString{}},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if err := EncodeSpannerStructFields(tc.source, tc.target); err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(tc.target, tc.wanted) {
				t.Errorf("%v does not match %v", tc.target, tc.wanted)
			}
		})
	}

}

func TestDecodeSpannerStructFields(t *testing.T) {
	type StrType string

	testcases := []struct {
		name   string
		source any
		target any
		wanted any
	}{
		{
			name:   "time.Time to openapi_types.Date",
			source: &struct{ D time.Time }{D: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)},
			target: &struct{ D openapi_types.Date }{},
			wanted: &struct{ D openapi_types.Date }{D: openapi_types.Date{Time: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)}},
		},
		{
			name:   "locally defined string type",
			source: &struct{ S spanner.NullString }{spanner.NullString{"hej", true}},
			target: &struct{ S *StrType }{},
			wanted: &struct{ S *StrType }{(*StrType)(StrP("hej"))},
		},
		{
			name:   "invalid nullstring to nil ptr to locally defined string type",
			source: &struct{ S spanner.NullString }{spanner.NullString{"hej", false}},
			target: &struct{ S *StrType }{},
			wanted: &struct{ S *StrType }{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if err := DecodeSpannerStructFields(tc.source, tc.target); err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(tc.target, tc.wanted) {
				t.Errorf("%v does not match %v", tc.target, tc.wanted)
			}
		})
	}

}

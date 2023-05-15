package common

import (
	"reflect"
	"testing"

	"cloud.google.com/go/spanner"
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

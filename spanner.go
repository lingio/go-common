package common

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"cloud.google.com/go/spanner"
)

// SpannerStructFieldNames returns a list with names of all struct fields.
// Fields with spanner tag value "-" will be ignored, any other tag value
// will be considered the new field name.
func SpannerStructFieldNames(s any) []string {
	var (
		t, _  = typeAndValueOfStruct(s)
		n     = t.NumField()
		names = make([]string, 0, n)
	)
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if tag := f.Tag.Get("spanner"); tag == "-" {
			continue
		} else if tag != "" {
			names = append(names, tag)
		} else {
			names = append(names, f.Name)
		}
	}
	return names
}

// DecodeSpannerStructFields copies all visible fields from source to target by
// struct field name. Fields will be copied using reflection. If field in
// source has spannerType tag containing `jsonstring`, the target field will be
// unmarshalled as json first.
//
//	var a User
//	var b DbUser
//	var c User
//	_ = EncodeDbStructFieldsInto(&a, &b)
//	_ = DecodeSpannerStructFields(&b, &c)
//	reflect.DeepEqual(a, c) // true
func DecodeSpannerStructFields(
	source any,
	target any,
) error {
	var (
		sourceType, sourceValue = typeAndValueOfStruct(source)
		targetType, targetValue = typeAndValueOfStruct(target)
	)

	var (
		sourceFields = reflect.VisibleFields(sourceType)
		targetFields = reflect.VisibleFields(targetType)
	)

	for _, sf := range sourceFields {
		sfv := sourceValue.FieldByIndex(sf.Index)
		// no need to copy zero values: "", 0, nil ptrs
		if !sfv.IsValid() || sfv.IsZero() {
			continue
		}

		var tf = findStructFieldByName(targetFields, sf.Name)

		// skip if target doesnt have field
		if tf.Type == nil || tf.Type.Kind() == reflect.Invalid {
			continue
		}

		tfv := targetValue.FieldByIndex(tf.Index)
		if sf.Tag.Get("spannerType") == "jsonstring" {
			var data []byte
			switch v := sfv.Interface().(type) {
			case string:
				data = []byte(v)
			case spanner.NullString:
				if !v.Valid {
					continue
				}
				data = []byte(v.StringVal)
			default:
				return fmt.Errorf("unknown type %T -> %T", sfv.Interface(), tfv.Interface())
			}

			if err := json.Unmarshal(data, tfv.Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		switch v := sfv.Interface().(type) {
		case int64, bool, string, time.Time:
			tfv.Set(sfv.Convert(tf.Type))
		case spanner.NullTime:
			if !v.Valid {
				continue
			}
			tfv.Set(reflect.ValueOf(&v.Time))
		case spanner.NullString:
			if !v.Valid {
				continue
			}
			tfv.Set(reflect.ValueOf(&v.StringVal))
		case spanner.NullFloat64:
			if !v.Valid {
				continue
			}
			tfv.Set(reflect.ValueOf(&v.Float64))
		case spanner.NullBool:
			if !v.Valid {
				continue
			}
			tfv.Set(reflect.ValueOf(&v.Bool))
		case spanner.NullInt64:
			if !v.Valid {
				continue
			}
			// assume tf.Type is a pointer to a value
			// tf.Type.Elem() is the type being pointed to
			valueType := tf.Type.Elem()
			int64AsValueType := reflect.ValueOf(v.Int64).Convert(valueType)
			// reflect.New(x) return type is PointerTo(x)
			// tfv.Elem() is the pointed-at value
			tfv.Set(reflect.New(valueType))
			tfv.Elem().Set(int64AsValueType)
		default:
			return fmt.Errorf("unknown type %T -> %T", sfv.Interface(), tfv.Interface())

		}
	}

	return nil
}

// EncodeSpannerStructFields copies all visible fields from source to target by
// struct field name. Fields will be copied using reflection. If field in
// target has spannerType tag containing `jsonstring`, the source field will be
// encoded as json first. The resulting data will be set as either `*string`
// or `string`, depending on the target field type.
//
//	type User struct {
//	   Age int
//		Inventory struct { /* complex */ }
//	}
//
//	type DbUser struct {
//	   Age int
//	   Inventory string `spannerType:"jsonstring"`
//	}
//
//	// This will copy `Age` and `Inventory` fields.
//	// `Inventory` will be json-encoded in DbUser.
//	var u User
//	var dbu DbUser
//	_ = EncodeSpannerStructFields(&u, &dbu)
func EncodeSpannerStructFields(
	source any,
	target any,
) error {
	var (
		sourceType, sourceValue = typeAndValueOfStruct(source)
		targetType, targetValue = typeAndValueOfStruct(target)
	)
	var (
		sourceFields = reflect.VisibleFields(sourceType)
		targetFields = reflect.VisibleFields(targetType)
	)

	for _, sf := range sourceFields {
		sfv := sourceValue.FieldByIndex(sf.Index)
		if !sfv.IsValid() || sfv.IsZero() {
			continue
		}

		tf := findStructFieldByName(targetFields, sf.Name)

		// skip if target doesnt have field
		if tf.Type == nil || tf.Type.Kind() == reflect.Invalid {
			continue
		}

		tfv := targetValue.FieldByIndex(tf.Index)

		// json encode path
		if tf.Tag.Get("spannerType") == "jsonstring" {
			data, err := json.Marshal(sfv.Interface())
			if err != nil {
				return err
			}

			strdata := string(data)
			switch tfv.Interface().(type) {
			case string:
				tfv.Set(reflect.ValueOf(strdata))
			case spanner.NullString:
				// note: the zero value of NullXxx is null so only write if valid
				tfv.Set(reflect.ValueOf(spanner.NullString{
					StringVal: strdata,
					Valid:     true,
				}))
			default:
				return fmt.Errorf("cannot store type %T -> %T", sfv.Interface(), tfv.Interface())
			}
			continue
		}

		// value copy path, with null wrapping
		switch v := sfv.Interface().(type) {
		case int, int32, bool, string, int64, float64, time.Time:
			tfv.Set(sfv.Convert(tfv.Type()))
		case *time.Time:
			tfv.Set(reflect.ValueOf(spanner.NullTime{
				Time:  *v,
				Valid: true,
			}))
		case *bool:
			tfv.Set(reflect.ValueOf(spanner.NullBool{
				Bool:  *v,
				Valid: true,
			}))
		case *string:
			tfv.Set(reflect.ValueOf(spanner.NullString{
				StringVal: *v,
				Valid:     true,
			}))
		case *int:
			tfv.Set(reflect.ValueOf(spanner.NullInt64{
				Int64: int64(*v),
				Valid: true,
			}))
		case *int64:
			tfv.Set(reflect.ValueOf(spanner.NullInt64{
				Int64: *v,
				Valid: true,
			}))
		case *float64:
			tfv.Set(reflect.ValueOf(spanner.NullFloat64{
				Float64: *v,
				Valid:   true,
			}))
		default:
			return fmt.Errorf("cannot copy type %T -> %T", sfv.Interface(), tfv.Interface())
		}
	}
	return nil
}

func typeAndValueOfStruct(x any) (reflect.Type, reflect.Value) {
	v := reflect.ValueOf(x)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	return v.Type(), v
}

func findStructFieldByName(fields []reflect.StructField, name string) reflect.StructField {
	for _, f := range fields {
		if name == f.Name {
			return f
		}
	}
	return reflect.StructField{}
}

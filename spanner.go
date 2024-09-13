package common

import (
	"context"

	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"

	// "encoding/json"
	"fmt"
	"reflect"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/goccy/go-json"
)

var (
	typeSpannerNullStr = reflect.TypeOf(spanner.NullString{})
	typeStrPtr         = reflect.TypeOf(StrP(""))
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

		tf := findStructFieldByName(targetFields, sf.Name)

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
		} else if sf.Type == reflect.TypeOf(time.Time{}) && tf.Type == reflect.TypeOf(openapi_types.Date{}) { // Convert from time.Time to openapi_types.Date
			tfv.Set(reflect.ValueOf(openapi_types.Date{Time: sfv.Interface().(time.Time)}))
			continue
		} else if sf.Type == typeSpannerNullStr && typeStrPtr.ConvertibleTo(tf.Type) {
			ns := sfv.Interface().(spanner.NullString)
			if !ns.Valid {
				continue
			}
			tfv.Set(reflect.ValueOf(&ns.StringVal).Convert(tf.Type))
			continue
		}

		switch v := sfv.Interface().(type) {
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
			if !sfv.CanConvert(tf.Type) {
				return fmt.Errorf("unknown type %T -> %T", sfv.Interface(), tfv.Interface())
			}
			tfv.Set(sfv.Convert(tf.Type))
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
//	   Inventory struct { /* complex */ }
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
		} else if sf.Type == reflect.TypeOf(openapi_types.Date{}) { // Convert from openapi_types.Date to time.Time
			tfv.Set(reflect.ValueOf(sfv.Interface().(openapi_types.Date).Time))
			continue
		} else if tf.Type == typeSpannerNullStr && sf.Type.ConvertibleTo(typeStrPtr) {
			// Convert from *string-compatible type to spanner.NullString
			// type Value string
			// type Data struct { Optional *Value }
			// type Encoded struct { Optional spanner.NullString }
			tfv.Set(reflect.ValueOf(spanner.NullString{
				StringVal: sfv.Convert(typeStrPtr).Elem().String(),
				Valid:     true,
			}))
			continue
		}

		// value copy path, with null wrapping
		switch v := sfv.Interface().(type) {
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
			if !sfv.CanConvert(tf.Type) {
				return fmt.Errorf("cannot copy type %T -> %T", sfv.Interface(), tfv.Interface())
			}
			tfv.Set(sfv.Convert(tf.Type))
		}
	}
	return nil
}

// SpannerReadTyped returns all rows in keySet from primary index,
// deserialized to struct T.
//
// Spanner keyset primer:
//
//	spanner.AllKeys()                        // fetch all rows in db
//	spanner.Key{"A", "B"}.AsPrefix()         // fetch all rows with primary index prefixed with {A,B}
//	spanner.KeySetFromKeys(spanner.Key{"A"}) // fetch one row with primary index = "A"
func SpannerReadTyped[T any](ctx context.Context, cli *spanner.Client, table string, keySet spanner.KeySet) ([]T, error) {
	return SpannerReadTypedWithOptions[T](ctx, cli, table, keySet, nil)
}

// SpannerReadTypedUsingIndex is identical to [SpannerReadTyped] but uses the
// provided index.
//
// Make sure all expected columns are stored in the index.
func SpannerReadTypedUsingIndex[T any](ctx context.Context, cli *spanner.Client, table, index string, keySet spanner.KeySet) ([]T, error) {
	return SpannerReadTypedWithOptions[T](ctx, cli, table, keySet, &spanner.ReadOptions{Index: index})
}

// SpannerReadTypedWithOptions returns all rows in keySet from primary index,
// deserialized to struct T.
//
// Spanner keyset primer:
//
//	spanner.AllKeys()                        // fetch all rows in db
//	spanner.Key{"A", "B"}.AsPrefix()         // fetch all rows with primary index prefixed with {A,B}
//	spanner.KeySetFromKeys(spanner.Key{"A"}) // fetch one row with primary index = "A"
//
// Read more about key sets: https://pkg.go.dev/cloud.google.com/go/spanner@v1.44.0#KeySet
func SpannerReadTypedWithOptions[T any](ctx context.Context, cli *spanner.Client, table string, keySet spanner.KeySet, opts *spanner.ReadOptions) ([]T, error) {
	txn := cli.ReadOnlyTransaction()
	defer txn.Close()

	var t T
	it := txn.ReadWithOptions(ctx, table, keySet, SpannerStructFieldNames(t), opts)
	return SpannerReadProjected(it, ProjectIdentity[T])
}

// SpannerReadTypedAndDecode returns all rows in keySet from the primary
// index, deserialized to struct I and then decoded into struct T using
// [DecodeSpannerStructFields].
//
// Spanner keyset primer:
//
//	spanner.AllKeys()                        // fetch all rows in db
//	spanner.Key{"A", "B"}.AsPrefix()         // fetch all rows with primary index prefixed with {A,B}
//	spanner.KeySetFromKeys(spanner.Key{"A"}) // fetch one row with primary index = "A"
func SpannerReadTypedAndDecode[I any, T any](ctx context.Context, cli *spanner.Client, table string, keySet spanner.KeySet) ([]T, error) {
	return SpannerReadTypedAndDecodeWithOptions[I, T](ctx, cli, table, keySet, nil)
}

// SpannerReadTypedAndDecodeUsingIndex is identical to
// [SpannerReadTypedAndDecode] but uses the provided index.
//
// Make sure all expected columns are stored in the index.
func SpannerReadTypedAndDecodeUsingIndex[I any, T any](ctx context.Context, cli *spanner.Client, table, index string, keySet spanner.KeySet) ([]T, error) {
	return SpannerReadTypedAndDecodeWithOptions[I, T](ctx, cli, table, keySet, &spanner.ReadOptions{Index: index})
}

// SpannerReadTypedAndDecodeWithOptions returns all rows in keySet from the
// primary index, deserialized to struct I and then decoded into struct T
// using [DecodeSpannerStructFields].
//
// Spanner keyset primer:
//
//	spanner.AllKeys()                        // fetch all rows in db
//	spanner.Key{"A", "B"}.AsPrefix()         // fetch all rows with primary index prefixed with {A,B}
//	spanner.KeySetFromKeys(spanner.Key{"A"}) // fetch one row with primary index = "A"
//
// Read more about key sets: https://pkg.go.dev/cloud.google.com/go/spanner@v1.44.0#KeySet
func SpannerReadTypedAndDecodeWithOptions[I any, T any](ctx context.Context, cli *spanner.Client, table string, keySet spanner.KeySet, opts *spanner.ReadOptions) ([]T, error) {
	txn := cli.ReadOnlyTransaction()
	defer txn.Close()

	var i I // intermediate
	it := txn.ReadWithOptions(ctx, table, keySet, SpannerStructFieldNames(i), opts)
	return SpannerReadProjected(it, ProjectDecoded[I, T])
}

// SpannerReadProjected returns projection `T -> P` of all rows in iterator.
//
// This is a low level helper. See [SpannerReadTyped] and [SpannerReadTypedAndDecode].
func SpannerReadProjected[T any, P any](ri *spanner.RowIterator, projection func(T) (P, error)) ([]P, error) {
	var (
		rows []P
	)

	err := ri.Do(
		func(r *spanner.Row) error {
			var i T
			if err := r.ToStruct(&i); err != nil {
				return Errorf(err, "spanner row to struct")
			}
			p, err := projection(i)
			if err != nil {
				return Errorf(err, "spanner struct projection")
			}
			rows = append(rows, p)
			return nil
		},
	)
	if err != nil {
		return nil, Errorf(err)
	}
	return rows, nil
}

// ProjectIdentity is a dummy projection
func ProjectIdentity[T any](i T) (T, error) { return i, nil }

// ProjectDecoded projects struct of type I to struct of type T using
// [DecodeSpannerStructFields].
func ProjectDecoded[I any, T any](i I) (T, error) {
	var t T
	if err := DecodeSpannerStructFields(i, &t); err != nil {
		return t, Errorf(err)
	}
	return t, nil
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

package common

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// DecodeSpannerStructFields copies all visible fields from source to target by
// struct field name. Fields will be copied using reflection. If field in
// source has spanner tag containing `asjson`, the target field will be
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
		sourceType, sourceValue = typeAndValueOf(source)
		targetType, targetValue = typeAndValueOf(target)
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
		if tf.Type.Kind() == reflect.Invalid {
			continue
		}

		tfv := targetValue.FieldByIndex(tf.Index)
		if !strings.Contains(sf.Tag.Get("spanner"), "asjson") {
			// just try to copy the value directly
			tfv.Set(sfv)
			continue
		}

		var data []byte
		switch sfv.Interface().(type) {
		case string:
			data = []byte(sfv.Interface().(string))
		case *string:
			data = []byte(*sfv.Interface().(*string))
		default:
			return fmt.Errorf("unknown type %T -> %T", sfv.Interface(), tfv.Interface())
		}

		if err := json.Unmarshal(data, tfv.Addr().Interface()); err != nil {
			return err
		}
	}

	return nil
}

// EncodeSpannerStructFields copies all visible fields from source to target by
// struct field name. Fields will be copied using reflection. If field in
// target has spanner tag containing `asjson`, the source field will be
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
//	   Inventory string `spanner:"inventory,asjson"`
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
		sourceType, sourceValue = typeAndValueOf(source)
		targetType, targetValue = typeAndValueOf(target)
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

		var tf = findStructFieldByName(targetFields, sf.Name)

		// skip if target doesnt have field
		if tf.Type.Kind() == reflect.Invalid {
			continue
		}

		tfv := targetValue.FieldByIndex(tf.Index)

		fmt.Println(tf.Name, sfv.Kind(), "->", tfv.Kind())
		if !strings.Contains(tf.Tag.Get("spanner"), "asjson") {
			// just try to copy the value directly
			tfv.Set(sfv)
			continue
		}

		data, err := json.Marshal(sfv.Interface())
		if err != nil {
			return err
		}

		strdata := string(data)
		switch tfv.Interface().(type) {
		case string:
			tfv.Set(reflect.ValueOf(strdata))
			continue
		case *string:
			tfv.Set(reflect.ValueOf(&strdata))
			continue
		}

		return fmt.Errorf("unknown type %T -> %T", sfv.Interface(), tfv.Interface())
	}
	return nil
}

func typeAndValueOf(x any) (reflect.Type, reflect.Value) {
	var (
		t = reflect.TypeOf(x)
		v = reflect.ValueOf(x)
	)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		v = v.Elem()
	}
	return t, v
}

func findStructFieldByName(fields []reflect.StructField, name string) reflect.StructField {
	for _, f := range fields {
		if name == f.Name {
			return f
		}
	}
	return reflect.StructField{}
}

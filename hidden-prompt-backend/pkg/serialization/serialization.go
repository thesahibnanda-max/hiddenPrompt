package serialization

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Deserializer interface {
	TextDeserialize(data string) error
}

type Serializer interface {
	TextSerialize() (string, error)
}

func Serialize(obj interface{}) (string, error) {
	return defaultEncoder.Encode(obj)
}

var defaultEncoder = &encoder{}

type encoder struct {
	cache sync.Map
}

type fieldInfo struct {
	index     int
	name      string
	omitempty bool
}

func (e *encoder) Encode(obj interface{}) (string, error) {
	if obj == nil {
		return "", nil
	}

	v := reflect.ValueOf(obj)

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", nil
		}
		v = v.Elem()
	}

	switch v.Kind() {

	case reflect.Struct:
		return e.encodeStruct(v)

	case reflect.Slice, reflect.Array:
		return e.encodeSlice(v)

	default:
		return e.encodeValue(v)
	}
}

func (e *encoder) encodeSlice(v reflect.Value) (string, error) {
	var parts []string

	for i := 0; i < v.Len(); i++ {

		s, err := e.Encode(v.Index(i).Interface())
		if err != nil {
			return "", err
		}

		parts = append(parts, s)
	}

	return "[" + strings.Join(parts, " ; ") + "]", nil
}

func (e *encoder) encodeStruct(v reflect.Value) (string, error) {
	fields := e.fields(v.Type())

	var parts []string

	for _, f := range fields {

		fv := v.Field(f.index)

		if f.omitempty && fv.IsZero() {
			continue
		}

		s, err := e.encodeValue(fv)
		if err != nil {
			return "", err
		}

		parts = append(parts, f.name+"="+s)
	}

	return strings.Join(parts, ", "), nil
}

func (e *encoder) fields(t reflect.Type) []fieldInfo {

	if v, ok := e.cache.Load(t); ok {
		return v.([]fieldInfo)
	}

	var out []fieldInfo

	for i := 0; i < t.NumField(); i++ {

		sf := t.Field(i)

		if sf.PkgPath != "" {
			continue
		}

		tag := sf.Tag.Get("text")

		if tag == "-" {
			continue
		}

		f := fieldInfo{
			index: i,
			name:  sf.Name,
		}

		if tag != "" {

			parts := strings.Split(tag, ",")

			if parts[0] != "" {
				f.name = parts[0]
			}

			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					f.omitempty = true
				}
			}
		}

		out = append(out, f)
	}

	e.cache.Store(t, out)

	return out
}

var (
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	timeType          = reflect.TypeOf(time.Time{})
)

func (e *encoder) encodeValue(v reflect.Value) (string, error) {

	for v.Kind() == reflect.Pointer {

		if v.IsNil() {
			return "", nil
		}

		v = v.Elem()
	}

	// custom serializer
	if s, ok := v.Interface().(Serializer); ok {
		return s.TextSerialize()
	}

	// time.Time -> time.DateTime
	if v.Type() == timeType {
		t := v.Interface().(time.Time)
		return t.Format(time.DateTime), nil
	}

	// encoding.TextMarshaler
	if v.Type().Implements(textMarshalerType) {
		b, err := v.Interface().(encoding.TextMarshaler).MarshalText()
		return string(b), err
	}

	if reflect.PointerTo(v.Type()).Implements(textMarshalerType) && v.CanAddr() {
		b, err := v.Addr().Interface().(encoding.TextMarshaler).MarshalText()
		return string(b), err
	}

	switch v.Kind() {

	case reflect.String:
		return v.String(), nil

	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil

	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:

		return strconv.FormatInt(v.Int(), 10), nil

	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:

		return strconv.FormatUint(v.Uint(), 10), nil

	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 32), nil

	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil

	case reflect.Slice, reflect.Array:

		var vals []string

		for i := 0; i < v.Len(); i++ {

			s, err := e.encodeValue(v.Index(i))
			if err != nil {
				return "", err
			}

			vals = append(vals, s)
		}

		return "[" + strings.Join(vals, ",") + "]", nil

	case reflect.Struct:
		return defaultEncoder.encodeStruct(v)

	default:
		return "", fmt.Errorf("unsupported type %s", v.Type())
	}
}

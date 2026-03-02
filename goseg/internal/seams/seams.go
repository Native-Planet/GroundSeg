package seams

import (
	"reflect"
	"unsafe"
)

func Merge[T any](base, overrides T) T {
	mergeValue(reflect.ValueOf(&base).Elem(), reflect.ValueOf(overrides))
	return base
}

func WithDefaults[T any](runtime, defaults T) T {
	return Merge(defaults, runtime)
}

func mergeValue(base, overrides reflect.Value) {
	if !base.IsValid() || !overrides.IsValid() {
		return
	}
	base = settableValue(base)
	if base.Kind() != reflect.Struct || overrides.Kind() != reflect.Struct {
		if isNonNil(overrides) && base.CanSet() {
			base.Set(overrides)
		}
		return
	}
	for i := 0; i < overrides.NumField(); i++ {
		baseField := settableValue(base.Field(i))
		overrideField := overrides.Field(i)
		if !baseField.IsValid() || !baseField.CanSet() {
			continue
		}
		if baseField.Kind() == reflect.Struct && overrideField.Kind() == reflect.Struct {
			mergeValue(baseField, overrideField)
			continue
		}
		if isNonNil(overrideField) {
			baseField.Set(overrideField)
		}
	}
}

func settableValue(v reflect.Value) reflect.Value {
	if v.CanSet() || !v.CanAddr() {
		return v
	}
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func isNonNil(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return !v.IsNil()
	default:
		return !v.IsZero()
	}
}

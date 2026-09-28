package api

import (
	"reflect"
	"strings"
)

func TrimAllStrings(v interface{}) {
	trimReflect(reflect.ValueOf(v))
}

func trimReflect(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			trimReflect(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			trimReflect(v.Field(i))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			trimReflect(v.Index(i))
		}
	case reflect.Map:
		if v.IsNil() {
			return
		}
		m := reflect.MakeMapWithSize(v.Type(), v.Len())
		for _, k := range v.MapKeys() {
			m.SetMapIndex(trimMapKey(k), trimMapValue(v.MapIndex(k)))
		}
		if v.CanSet() {
			v.Set(m)
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(strings.TrimSpace(v.String()))
		}
	}
}

// trimMapKey  map
func trimMapKey(k reflect.Value) reflect.Value {
	if k.Kind() == reflect.String {
		ck := reflect.New(k.Type()).Elem()
		ck.SetString(strings.TrimSpace(k.String()))
		return ck
	}
	return k
}

// trimMapValue  map MapIndex
func trimMapValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.String {
		cp := reflect.New(v.Type()).Elem()
		cp.SetString(strings.TrimSpace(v.String()))
		return cp
	}
	cp := reflect.New(v.Type()).Elem()
	cp.Set(v)
	trimReflect(cp)
	return cp
}
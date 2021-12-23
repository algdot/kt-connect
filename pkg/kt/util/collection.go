package util

import "reflect"

// Contains check whether obj exist in target, the type of target can be an array, slice or map
func Contains(obj interface{}, target interface{}) bool {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true
		}
	}
	return false
}

func MapContains(subset, fullset map[string]string) bool {
	for sk, sv := range subset {
		find := false
		for fk, fv := range fullset {
			if sk == fk && sv == fv {
				find = true
				break
			}
		}
		if !find {
			return false
		}
	}
	return true
}

func MapPut(m map[string]string, k, v string) {
	if m == nil {
		m = make(map[string]string)
	}
	m[k] = v
}

func MergeMap(m1 map[string]string, m2 map[string]string) map[string]string {
	cp := make(map[string]string)
	for key, value := range m1 {
		cp[key] = value
	}
	for key, value := range m2 {
		cp[key] = value
	}
	return cp
}

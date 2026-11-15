package kit

import (
	"crypto/subtle"
	"reflect"
)

// Eq compares two strings in constant time, for secrets: plain == leaks a timing side channel byte by byte.
func Eq(s1, s2 string) bool {
	b1 := []byte(s1)
	b2 := []byte(s2)
	return subtle.ConstantTimeEq(int32(len(b1)), int32(len(b2))) == 1 && subtle.ConstantTimeCompare(b1, b2) == 1 //nolint:gosec // that's ok
}

// IsNil reports whether v is nil, including an interface wrapping a typed-nil pointer that == nil would miss.
func IsNil(i any) bool {
	// standard nil check
	if i == nil {
		return true
	}

	// special case: interfaces holding nil pointers (e.g. var x *int = nil; var y any = x)
	rv := reflect.ValueOf(i)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}

	// check for other kinds that can be nil
	switch rv.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice,
		reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}

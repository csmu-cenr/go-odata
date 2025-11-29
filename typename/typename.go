package typename

import (
	"fmt"
	"reflect"
)

// TypeName returns a readable string name for the concrete type T.
// It works for pointers, slices, maps, arrays, and anonymous types.
func ShortTypeName[T any]() string {
	var zero T
	return shortTypeNameOf(reflect.TypeOf(zero))
}

// TypeName returns a readable string name for the concrete type T.
// It works for pointers, slices, maps, arrays, and anonymous types.
func TypeName[T any]() string {
	var zero T
	return typeNameOf(reflect.TypeOf(zero))
}

func typeNameOf(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}

	switch t.Kind() {

	case reflect.Pointer:
		return "*" + typeNameOf(t.Elem())

	case reflect.Slice:
		return "[]" + typeNameOf(t.Elem())

	case reflect.Array:
		return "[" + fmt.Sprint(t.Len()) + "]" + typeNameOf(t.Elem())

	case reflect.Map:
		return "map[" + typeNameOf(t.Key()) + "]" + typeNameOf(t.Elem())

	case reflect.Chan:
		return "chan " + typeNameOf(t.Elem())

	case reflect.Func:
		return "func" // keeping it simple; could be expanded

	case reflect.Struct:
		// Named struct: use package path + name
		if t.Name() != "" {
			if t.PkgPath() != "" {
				return t.PkgPath() + "." + t.Name()
			}
			return t.Name()
		}
		// Anonymous struct
		return t.String()

	default:
		// Basic types or named types
		if t.Name() != "" {
			if t.PkgPath() != "" {
				return t.PkgPath() + "." + t.Name()
			}
			return t.Name()
		}

		// Fallback (e.g. interface{}, unnamed types)
		return t.String()
	}
}

func shortTypeNameOf(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}

	switch t.Kind() {

	case reflect.Pointer:
		return "*" + shortTypeNameOf(t.Elem())

	case reflect.Slice:
		return "[]" + shortTypeNameOf(t.Elem())

	case reflect.Array:
		return "[" + fmt.Sprint(t.Len()) + "]" + shortTypeNameOf(t.Elem())

	case reflect.Map:
		return "map[" + shortTypeNameOf(t.Key()) + "]" + shortTypeNameOf(t.Elem())

	case reflect.Chan:
		return "chan " + shortTypeNameOf(t.Elem())

	case reflect.Func:
		return "func" // keeping it simple; could be expanded

	case reflect.Struct:
		// Named struct: use package path + name
		if t.Name() != "" {
			return t.Name()
		}
		// Anonymous struct
		return t.String()

	default:
		// Basic types or named types
		if t.Name() != "" {
			return t.Name()
		}

		// Fallback (e.g. interface{}, unnamed types)
		return t.String()
	}
}

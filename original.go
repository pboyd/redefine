//go:build amd64

package redefine

import "reflect"

// Original returns a function with the same behavior as the original version
// of the function. If the function has not been redefined the original version
// if the passed function to that will be returned.
//
// If the original function cannot be found for any reason Original returns nil.
//
// Technically, this returns a copy of the original that's been relocated and
// had relative addresses adjusted. This process may introduce problems.
func Original[T any](fn T) T {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return *((*T)(nil))
	}

	mu.RLock()
	defer mu.RUnlock()

	cloned, ok := redefined[fnv.Pointer()]
	if !ok {
		// Not redefined, so return the original func.
		return fn
	}

	if clonedType, ok := cloned.(*clonedFunc[T]); ok {
		return clonedType.Func
	}

	return *((*T)(nil))
}

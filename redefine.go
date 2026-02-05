package redefine

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

var mu sync.RWMutex

var redefined = map[uintptr]any{}

// Func redefines fn with newFn. An error will be returned if fn or newFn are
// not function pointers.
//
// Note that Func only modifies non-inlined functions. Anywhere that fn has
// been inlined will continue with the old behavior. If possible, add a
// noinline directive:
//
//	//go:noinline
//	func myfunc() {
//		...
//	}
//
// Other limitations that might be addressed one day:
//   - Generic functions cannot be redefined
//   - newFn cannot be a closure (anonymous functions are fine, but it will crash
//     if you attempt to use data from the stack)
func Func[T any](fn, newFn T) error {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func || fnv.IsNil() {
		return fmt.Errorf("not a function, kind: %v", fnv.Kind())
	}
	newFnv := reflect.ValueOf(newFn)
	if newFnv.Kind() != reflect.Func || newFnv.IsNil() {
		return fmt.Errorf("not a function, kind: %v", newFnv.Kind())
	}

	return unsafeFunc(fn, newFn)
}

// Method redefines a method of an object. The same caveats from Func apply
// here, with the new wrinkle that newFn must be a method on a type equivalent
// to the original type. For example:
//
//	type myCustomType otherpackage.Type
//
// Any other type for the instance of newFn will likely lead to very
// troublesome bugs because the code compiled for newFn will be operating on
// the memory for the instance of fn.
func Method(fn, newFn any) error {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return fmt.Errorf("not a function, kind: %v", fnv.Kind())
	}
	newFnv := reflect.ValueOf(newFn)
	if newFnv.Kind() != reflect.Func {
		return fmt.Errorf("not a function, kind: %v", newFnv.Kind())
	}

	diff := diffFuncs(fnv, newFnv)

	// Ignore differences in the first argument if they are the same kind and size.
	if len(diff.In) > 0 {
		if diff.In[0] != nil {
			ta := diff.In[0].A
			tb := diff.In[0].B

			if ta.Kind() == tb.Kind() {
				// If the types are pointers then check the
				// size of what they point to instead of the
				// size of the pointers.
				if ta.Kind() == reflect.Pointer {
					ta = ta.Elem()
					tb = tb.Elem()
				}

				if ta.Size() == tb.Size() {
					diff.In[0] = nil
				}
			}

		}
	}

	if err := diff.Error(); err != nil {
		return fmt.Errorf("function signatures do not match: %w", err)
	}

	return unsafeFunc(fnv.Interface(), newFnv.Interface())
}

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

// Restore reverses the effect of redefining a method.
func Restore[T any](fn T) error {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return fmt.Errorf("not a function, kind: %v", fnv.Kind())
	}

	mu.Lock()
	defer mu.Unlock()

	cloned, ok := redefined[fnv.Pointer()]
	if !ok {
		// Not redefined, this is a no-op
		return nil
	}

	clonedType, ok := cloned.(*clonedFunc[T])
	if !ok {
		return fmt.Errorf("unknown function type")
	}

	code, err := funcSlice(fn)
	if err != nil {
		return err
	}
	if len(code) != len(clonedType.originalCode) {
		fmt.Errorf("func length mismatch %d != %d", len(code), len(clonedType.originalCode))
	}

	err = mprotect(code, mprotectRWX)
	if err != nil {
		return err
	}
	defer mprotect(code, mprotectRX)

	copy(code, clonedType.originalCode)

	clonedType.Free()
	delete(redefined, fnv.Pointer())

	return nil
}

// unsafeFunc redefines a function after the safety checks.
func unsafeFunc[T any](fn, newFn T) error {
	code, err := funcSlice(fn)
	if err != nil {
		return err
	}

	// Locked to prevent simultaneous writes to the map and competing
	// mprotect calls
	mu.Lock()
	defer mu.Unlock()

	addr := reflect.ValueOf(fn).Pointer()
	if _, ok := redefined[addr]; !ok {
		redefined[addr], err = cloneFunc(fn)
		if err != nil {
			// TODO: Should this be fatal?
			return err
		}
	}

	err = mprotect(code, mprotectRWX)
	if err != nil {
		return err
	}
	defer mprotect(code, mprotectRX)

	// TODO: If the size the new code is less than the size of the old code
	// then why not just copy it?
	return insertJump(code, reflect.ValueOf(newFn).Pointer())
}

// funcSlice returns a slice containing the machine instructions for a function.
func funcSlice(fn any) ([]byte, error) {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return nil, fmt.Errorf("not a function, kind: %v", fnv.Kind())
	}

	entry := fnv.Pointer()

	// To find the length, look at the offsets of every function and find
	// the one that comes immediately after this one.

	// TODO: Is there a better way to do this?
	//    - ftab seems to be ordered so could it find the next entry that way?
	//    - is the info stored somewhere more conveniently in datap?

	info := findfunc(entry)
	funcOffset := uint32(entry - info.datap.text)
	length := uint32(info.datap.etext - entry)

	for _, ft := range info.datap.ftab {
		// Does this function come before the one we're looking for?
		if ft.entryoff <= funcOffset {
			continue
		}

		// Is the distance between these two functions less than what we've seen before?
		testLength := ft.entryoff - funcOffset
		if testLength < length {
			length = testLength
		}
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(entry)), length), nil
}

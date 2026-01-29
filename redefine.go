package redefine

import (
	"fmt"
	"reflect"
	"unsafe"
)

// Func redefines fn with newFn. An error will be returned if fn or newFn are
// not function pointers or if their signatures do not match.
//
// Note that Func only modifies non-inlined functions. Anywhere that fn has
// been inlined it will continue with the old behavior. If possible, add a
// noinline directive:
//
//	//go:noinline
//	func myfunc() {
//		...
//	}
func Func(fn, newFn any) error {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return fmt.Errorf("not a function, kind: %v", fnv.Kind())
	}
	newFnv := reflect.ValueOf(newFn)
	if newFnv.Kind() != reflect.Func {
		return fmt.Errorf("not a function, kind: %v", newFnv.Kind())
	}
	diff := diffFuncs(fnv, newFnv)
	if err := diff.Error(); err != nil {
		return fmt.Errorf("function signatures do not match: %w", err)
	}

	return unsafeFunc(fnv, newFnv)
}

// Method redefines a method of an object type. The same caveats from Func
// apply here, with the new wrinkle that newFn must be a method on a type
// equivalent to the original type. For example:
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

	return unsafeFunc(fnv, newFnv)
}

// unsafeFunc redefines a function without the safety checks.
func unsafeFunc(fnv, newFnv reflect.Value) error {
	code, err := funcSlice(fnv)
	if err != nil {
		return err
	}

	err = mprotect(code, mprotectRWX)
	if err != nil {
		return err
	}
	defer mprotect(code, mprotectRX)

	return insertJump(code, newFnv.Pointer())
}

// funcSlice returns a slice containing the machine instructions for a function.
func funcSlice(fn reflect.Value) ([]byte, error) {
	if fn.Kind() != reflect.Func {
		return nil, fmt.Errorf("not a function, kind: %v", fn.Kind())
	}

	entry := fn.Pointer()

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

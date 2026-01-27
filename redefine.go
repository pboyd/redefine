package redefine

import (
	"fmt"
	"reflect"
	"unsafe"
)

// Func redefines fn with newFn. An error will be returned if fn or newFn are
// not function pointers or if their signatures do not match.
//
// Note that if fn has been inlined this will silently fail. If possible, add a
// noinline directive to work-around this problem:
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
	if !funcsAreEqual(fnv, newFnv) {
		return fmt.Errorf("function signatures do not match")
	}

	code, err := funcSlice(fnv)
	if err != nil {
		return err
	}

	err = mprotect(code, mprotectRWX)
	if err != nil {
		return err
	}
	defer mprotect(code, mprotectRX)

	newFnEntry := newFnv.Pointer()
	if err != nil {
		return err
	}

	return insertJump(code, newFnEntry)
}

func funcsAreEqual(a, b reflect.Value) bool {
	at := a.Type()
	bt := b.Type()
	if at.NumIn() != bt.NumIn() {
		return false
	}
	if at.NumOut() != bt.NumOut() {
		return false
	}

	for i := 0; i < at.NumIn(); i++ {
		if at.In(i) != bt.In(i) {
			return false
		}
	}

	for i := 0; i < at.NumOut(); i++ {
		if at.Out(i) != bt.Out(i) {
			return false
		}
	}

	return true
}

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

	var buf []byte
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sliceHeader.Data = entry
	sliceHeader.Len = int(length)
	sliceHeader.Cap = int(length)
	return buf, nil
}

//go:build amd64

package redefine

import "unsafe"

func _cloneFunc[T any](fn T, originalCode []byte) (*clonedFunc[T], error) {
	originalCode, err := funcSlice(fn)
	if err != nil {
		return nil, err
	}

	//fmt.Println(disassemble(originalCode))

	cloneAllocator.BeginMutate()
	defer cloneAllocator.EndMutate()

	newCode, err := cloneAllocator.Allocate(len(originalCode))
	if err != nil {
		return nil, err
	}

	newCode, err = relocateFunc(originalCode, newCode)
	if err != nil {
		return nil, err
	}

	//fmt.Println(disassemble(newCode))

	// This seems too complicated. The idea is to take our newly allocated
	// buffer of machine instructions and convince Go that it's really a
	// function pointer of type T.
	codeData := unsafe.SliceData(newCode)
	cf := clonedFunc[T]{
		clonedCode: newCode,
		// Keep a reference to codeData so it stays around.
		ref: &codeData,
	}
	cf.Func = *(*T)(unsafe.Pointer(uintptr(unsafe.Pointer(&cf.ref))))

	return &cf, nil
}

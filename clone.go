package redefine

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/pboyd/malloc"
)

// CloneFunc makes a copy of a function.
func CloneFunc[T any](fn T) (*ClonedFunc[T], error) {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return nil, fmt.Errorf("not a function, kind: %v", fnv.Kind())
	}

	originalCode, err := funcSlice(fnv)
	if err != nil {
		return nil, err
	}

	//fmt.Println(disassemble(originalCode))

	// FIXME: It would be nice if relocateFunc could write directly to the
	// buffer from the allocator. But the allocator needs a fixed size and
	// don't know it early enough.
	relocatableCode, err := relocateFunc(originalCode)
	if err != nil {
		return nil, err
	}

	newCode, err := cloneAllocator.Allocate(len(relocatableCode))
	if err != nil {
		return nil, err
	}
	copy(newCode, relocatableCode)

	//fmt.Println(disassemble(newCode))

	// This seems too complicated. The idea is to take our newly allocated
	// buffer of machine instructions and convince Go that it's really a
	// function pointer of type T.
	codeData := unsafe.SliceData(newCode)
	cf := ClonedFunc[T]{
		code: newCode,
		// Keep a reference to codeData so it stays around.
		ref: &codeData,
	}
	cf.Func = *(*T)(unsafe.Pointer(uintptr(unsafe.Pointer(&cf.ref))))

	return &cf, nil
}

type allocator struct {
	*malloc.Arena
	mu       sync.Mutex
	initOnce sync.Once
}

func (a *allocator) init() error {
	var err error
	a.initOnce.Do(func() {
		var buf []byte
		// FIXME: The amount of memory to allocate should be configurable
		buf, err = mmap(1024*1024, mprotectRWX)
		if err != nil {
			return
		}

		a.Arena = malloc.NewArenaAt(buf)
		if a.Arena == nil {
			err = errors.New("unable to initialize arena")
			return
		}
	})
	return err
}

func (a *allocator) Allocate(size int) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	err := a.init()
	if err != nil {
		return nil, fmt.Errorf("error initializing allocator: %w", err)
	}

	return malloc.MallocSlice[byte](a.Arena, size)
}

func (a *allocator) Free(buf []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()

	malloc.FreeSlice[byte](a.Arena, buf)
}

var cloneAllocator = &allocator{}

// ClonedFunc holds a copy of a function.
type ClonedFunc[T any] struct {
	Func T

	// The data for this slice is allocated in the mmap page and managed by
	// allocator. Keep a reference in order to free it.
	code []byte
	ref  **byte
}

// Free releases the memory associated with the cloned function.
func (cf *ClonedFunc[T]) Free() {
	cloneAllocator.Free(cf.code)

	cf.code = nil
	*cf.ref = nil
	cf.ref = nil
}

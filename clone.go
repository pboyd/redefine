package redefine

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/pboyd/malloc"
)

// cloneFunc makes a copy of a function that persists after the original
// function has been modified.
func cloneFunc[T any](fn T) (*clonedFunc[T], error) {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return nil, fmt.Errorf("not a function, kind: %v", fnv.Kind())
	}

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

	// Make a copy of the code so that no matter what it can be restored.
	cf.originalCode = make([]byte, len(originalCode))
	copy(cf.originalCode, originalCode)

	return &cf, nil
}

type allocator struct {
	*malloc.Arena
	mprotect func(int) error
	mu       sync.Mutex
	initOnce sync.Once
	mutable  bool
}

func (a *allocator) init(startSize int) error {
	var err error
	a.initOnce.Do(func() {
		be := malloc.MmapBackend(mprotectExec, map_32bit)
		if protBE, ok := be.(malloc.ProtectedArenaBackend); ok {
			a.mprotect = protBE.Protect
		} else {
			a.mprotect = func(int) error {
				return nil
			}
		}

		a.Arena = malloc.NewArena(uint64(startSize), malloc.Backend(be))
		if a.Arena == nil {
			err = errors.New("unable to initialize arena")
			return
		}
		a.mutable = true
	})
	return err
}

func (a *allocator) BeginMutate() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Note that BeginMutate can be called before the initial allocation.

	if a.mprotect == nil || a.mutable {
		return nil
	}

	err := a.mprotect(mprotectRWX)
	if err == nil {
		a.mutable = true
	}
	return err
}

func (a *allocator) EndMutate() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.mutable {
		return nil
	}

	err := a.mprotect(mprotectRX)
	if err == nil {
		a.mutable = false
	}
	return err
}

func (a *allocator) Allocate(size int) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	err := a.init(size)
	if err != nil {
		return nil, fmt.Errorf("error initializing allocator: %w", err)
	}

	if !a.mutable {
		panic("Allocate called in immutable state")
	}

	return malloc.MallocSlice[byte](a.Arena, size)
}

func (a *allocator) Free(buf []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.mutable {
		panic("Free called in immutable state")
	}

	malloc.FreeSlice(a.Arena, buf)
}

var cloneAllocator = &allocator{}

// clonedFunc holds a copy of a function.
type clonedFunc[T any] struct {
	Func T

	// The data for this slice is allocated in the mmap page and managed by
	// the cloneAllocator. Keep a reference in order to free it.
	clonedCode []byte
	ref        **byte

	originalCode []byte
}

// Free releases the memory associated with the cloned function.
func (cf *clonedFunc[T]) Free() {
	cloneAllocator.BeginMutate()
	defer cloneAllocator.EndMutate()

	cloneAllocator.Free(cf.clonedCode)

	cf.clonedCode = nil
	*cf.ref = nil
	cf.ref = nil
	cf.originalCode = nil
}

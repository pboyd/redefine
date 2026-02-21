package redefine

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"sync"
	"syscall"
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

	cacheflush(newCode)

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
		var be malloc.ArenaBackend
		be, err = initMallocBackend()
		if err != nil {
			return
		}

		a.Arena = malloc.NewArena(uint64(startSize), malloc.Backend(be))
		if a.Arena == nil {
			err = errors.New("unable to initialize arena")
			return
		}

		if protBE, ok := be.(malloc.ProtectedArenaBackend); ok {
			a.mprotect = protBE.Protect
		} else {
			// No real mprotect for some reason. This shouldn't
			// really happen, but continue with a no-op mprotect.
			a.mprotect = func(int) error { return nil }
		}
		a.mutable = true
	})
	return err
}

// The lowest address to consider for our cloned functions.
const absMinAddress = 0x100000

func initMallocBackend() (malloc.ArenaBackend, error) {
	var text, etext uintptr
	var end uintptr
	pc, _, _, _ := runtime.Caller(0)
	datap := findfunc(pc).datap
	if datap != nil {
		text = datap.text
		etext = datap.etext
		end = datap.end
	}
	if text == 0 || etext == 0 || end == 0 {
		return nil, fmt.Errorf("failed to find moduledata")
	}

	pageSize := uintptr(syscall.Getpagesize())

	// Calculate the virtual memory reservation size. This amount
	// is reserved up-front, but pages are only committed as
	// needed.
	//
	// Use the size of the existing text segment so there's enough space to
	// clone every statically-linked function.
	size := (etext - text + pageSize - 1) &^ (pageSize - 1)

	// Cloned functions need to be near the existing text and data
	// segments so that they can be reached by the same
	// instructions that the original function used. There's often enough
	// space right before the text segment but that's not guaranteed
	// (particularly when buildmode=pie).

	// The minimum acceptable address is where the first
	// instruction in the code segment can still reach the final
	// address before end. These are unsigned so watch for wrap-around.
	minAddress := end - maxCloneDistance
	if minAddress > end || minAddress < absMinAddress {
		minAddress = absMinAddress
	}
	for addr := text - pageSize - size; addr >= minAddress; addr -= 0x100000 {
		be, err := malloc.VirtBackend(size, malloc.MmapAddr(addr), malloc.MmapProt(mprotectExec), malloc.MmapFlags(_MAP_FIXED_NOREPLACE))
		if err == nil {
			return be, nil
		}
	}

	// Nothing was found before the text segment, repeat the process for
	// the space after end.
	maxAddress := text + maxCloneDistance - size
	if maxAddress < text {
		maxAddress = math.MaxUint
	}
	for addr := end; addr <= maxAddress; addr += 0x100000 {
		be, err := malloc.VirtBackend(size, malloc.MmapAddr(addr), malloc.MmapProt(mprotectExec), malloc.MmapFlags(_MAP_FIXED_NOREPLACE))
		if err == nil {
			return be, nil
		}
	}

	return nil, errors.New("no suitable virtual memory space found")
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

	if cf.clonedCode != nil {
		cloneAllocator.Free(cf.clonedCode)
	}

	cf.clonedCode = nil
	if cf.ref != nil {
		*cf.ref = nil
		cf.ref = nil
	}
	cf.originalCode = nil
}

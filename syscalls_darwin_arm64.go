//go:build darwin && arm64

package redefine

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	mprotectExec = 0 // unused: allocator does not use MAP_JIT, no simultaneous W+X needed
	mprotectRX   = syscall.PROT_READ | syscall.PROT_EXEC
	// W^X: BeginMutate uses RW (not RWX); execute permission is restored by EndMutate.
	mprotectRWX = syscall.PROT_READ | syscall.PROT_WRITE
)

// makeRWX and makeRX are no-ops on darwin/arm64. MAP_JIT text patching is
// handled by applyCodeJump / applyCodeCopy which bracket the JIT write in C.
func makeRWX(buf []byte) error { return nil }
func makeRX(buf []byte) error  { return nil }

// applyCodeCopy writes src into dst on MAP_JIT pages via writeJITCode, which
// performs the pthread_jit_write_protect_np toggle and I-cache flush entirely
// in C so that the return to Go is always in execute mode.
func applyCodeCopy(dst, src []byte) error {
	writeJITCode(dst[:len(src)], src)
	return nil
}

// applyCodeJump encodes a B instruction targeting dest for execution at
// code's address, then writes it to code via writeJITCode.
func applyCodeJump(code []byte, dest uintptr) error {
	srcAddr := uintptr(unsafe.Pointer(unsafe.SliceData(code)))
	offset := int64(dest) - int64(srcAddr)
	if offset < -(1<<27) || offset >= (1<<27) {
		return fmt.Errorf("B target out of range: %d bytes exceeds 128MiB", offset)
	}

	// Build the full replacement in a temp buffer (correct for srcAddr), then
	// atomically write it to the live MAP_JIT page via jit_memcpy.
	tmp := make([]byte, len(code))
	encodeB(tmp, int32(offset))
	// zero-pad the rest (same as insertJump does)
	for i := 4; i < len(tmp); i++ {
		tmp[i] = 0
	}
	writeJITCode(code, tmp)
	return nil
}

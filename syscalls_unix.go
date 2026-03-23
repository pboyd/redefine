//go:build linux || (darwin && amd64) || openbsd || netbsd || freebsd

package redefine

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/pboyd/redefine/internal/cacheflush"
	"golang.org/x/sys/unix"
)

const (
	mprotectExec = syscall.PROT_EXEC
	mprotectRX   = syscall.PROT_READ | syscall.PROT_EXEC
	mprotectRWX  = syscall.PROT_READ | syscall.PROT_WRITE | syscall.PROT_EXEC
)

func makeRWX(buf []byte) error {
	return mprotect(buf, mprotectRWX)
}

func makeRX(buf []byte) error {
	return mprotect(buf, mprotectRX)
}

func applyCodeCopy(dst, src []byte) error {
	if err := makeRWX(dst); err != nil {
		return err
	}
	defer makeRX(dst)
	copy(dst[:len(src)], src)
	cacheflush.Flush(dst)
	return nil
}

func applyCodeJump(code []byte, dest uintptr) error {
	if err := makeRWX(code); err != nil {
		return fmt.Errorf("mprotect: %w", err)
	}
	defer makeRX(code)
	if err := insertJump(code, dest); err != nil {
		return err
	}
	cacheflush.Flush(code)
	return nil
}

func mprotect(buf []byte, flags int) error {
	pageSize := syscall.Getpagesize()

	addr := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))

	// Round address down to page boundary.
	pageStart := addr &^ (uintptr(pageSize) - 1)

	// Round up to cover complete pages.
	regionSize := (int(addr-pageStart) + cap(buf) + pageSize - 1) &^ (pageSize - 1)

	return unix.Mprotect(unsafe.Slice((*byte)(unsafe.Pointer(pageStart)), regionSize), flags)
}

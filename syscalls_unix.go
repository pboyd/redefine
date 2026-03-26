//go:build linux || (darwin && amd64) || openbsd || netbsd || freebsd

package redefine

import (
	"syscall"
	"unsafe"

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

func mprotect(buf []byte, flags int) error {
	pageSize := syscall.Getpagesize()

	addr := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))

	// Round address down to page boundary.
	pageStart := addr &^ (uintptr(pageSize) - 1)

	// Round up to cover complete pages.
	regionSize := (int(addr-pageStart) + cap(buf) + pageSize - 1) &^ (pageSize - 1)

	return unix.Mprotect(unsafe.Slice((*byte)(unsafe.Pointer(pageStart)), regionSize), flags)
}

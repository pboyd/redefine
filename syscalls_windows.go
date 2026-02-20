//go:build windows

package redefine

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	mprotectExec = windows.PAGE_EXECUTE
	mprotectRX   = windows.PAGE_EXECUTE_READ
	mprotectRWX  = windows.PAGE_EXECUTE_READWRITE
)

func mprotect(buf []byte, flags int) error {
	pageSize := syscall.Getpagesize()

	addr := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))

	// Round address down to page boundary.
	pageStart := addr &^ (uintptr(pageSize) - 1)

	// Round up to cover complete pages.
	regionSize := (int(addr-pageStart) + cap(buf) + pageSize - 1) &^ (pageSize - 1)

	var oldFlags uint32
	return windows.VirtualProtect(pageStart, uintptr(regionSize), uint32(flags), &oldFlags)
}

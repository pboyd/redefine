//go:build windows

package redefine

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/pboyd/redefine/internal/cacheflush"
	"golang.org/x/sys/windows"
)

const (
	mprotectExec = windows.PAGE_EXECUTE
	mprotectRX   = windows.PAGE_EXECUTE_READ
	mprotectRWX  = windows.PAGE_EXECUTE_READWRITE
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

	var oldFlags uint32
	return windows.VirtualProtect(pageStart, uintptr(regionSize), uint32(flags), &oldFlags)
}

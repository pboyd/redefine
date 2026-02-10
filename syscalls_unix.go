package redefine

import (
	"syscall"
	"unsafe"
)

const (
	mprotectRX  = syscall.PROT_READ | syscall.PROT_EXEC
	mprotectRWX = syscall.PROT_READ | syscall.PROT_WRITE | syscall.PROT_EXEC
)

func mprotect(buf []byte, flags int) error {
	pageSize := syscall.Getpagesize()

	addr := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))

	// Round address down to page boundary.
	pageStart := addr &^ (uintptr(pageSize) - 1)

	// Round up to cover complete pages.
	regionSize := (int(addr-pageStart) + cap(buf) + pageSize - 1) &^ (pageSize - 1)

	return syscall.Mprotect(unsafe.Slice((*byte)(unsafe.Pointer(pageStart)), regionSize), flags)
}

// Not defined for GOOS=darwin for some reason
const map_32bit = 0x40

func mmap(size int, prot int) ([]byte, error) {
	pageSize := syscall.Getpagesize()
	size = (size + pageSize - 1) &^ (pageSize - 1)

	return syscall.Mmap(-1, 0, size, prot, syscall.MAP_PRIVATE|syscall.MAP_ANON|map_32bit)
}

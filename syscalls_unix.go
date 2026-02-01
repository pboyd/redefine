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
	addr := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))

	pageSize := syscall.Getpagesize()

	// Round address down to page boundary.
	// Example: addr=4196 with pageSize=4096 becomes 4096.
	pageStart := addr - (addr % uintptr(pageSize))

	// Calculate how many bytes from pageStart we need to cover.
	// This includes the offset from pageStart to addr, plus the requested length.
	offsetWithinPage := int(addr - pageStart)
	totalBytes := offsetWithinPage + cap(buf)

	// Round up to cover complete pages.
	regionSize := (totalBytes + pageSize - 1) / pageSize * pageSize

	// Convert the memory region to a byte slice for mprotect.
	region := unsafe.Slice((*byte)(unsafe.Pointer(pageStart)), regionSize)

	return syscall.Mprotect(region, flags)
}

func mmap(size int, prot int) ([]byte, error) {
	pageSize := syscall.Getpagesize()
	size = (size + pageSize - 1) / pageSize * pageSize

	return syscall.Mmap(-1, 0, size, prot, syscall.MAP_PRIVATE|syscall.MAP_ANON)
}

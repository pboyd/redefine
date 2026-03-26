//go:build darwin && arm64

package static

import (
	"fmt"
	"unsafe"

	"github.com/pboyd/redefine/internal/mach"
	"golang.org/x/sys/unix"
)

func (s *Info) getWriteOffset() (uintptr, error) {
	if s.writeOffset != 0 {
		return s.writeOffset, nil
	}

	text := s.datap.text & pageMask
	etext := (s.datap.etext + pageSize - 1) & pageMask
	size := etext - text

	newText, err := unix.MmapPtr(-1, 0, nil, size,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_ANON|unix.MAP_PRIVATE,
	)
	if err != nil {
		return 0, fmt.Errorf("mmap: %w", err)
	}

	src := unsafe.Slice((*byte)(unsafe.Pointer(text)), size)
	dest := unsafe.Slice((*byte)(newText), size)

	copy(dest, src)

	err = unix.Mprotect(dest, unix.PROT_READ|unix.PROT_EXEC)
	if err != nil {
		unix.MunmapPtr(newText, size)
		return 0, fmt.Errorf("mprotect r-x: %w", err)
	}

	_, err = mach.VmRemap(text, uintptr(newText), size)
	if err != nil {
		unix.MunmapPtr(newText, size)
		return 0, fmt.Errorf("vmRemap: %w", err)
	}

	err = unix.Mprotect(dest, unix.PROT_READ|unix.PROT_WRITE)
	if err != nil {
		unix.MunmapPtr(newText, size)
		return 0, fmt.Errorf("mprotect rw-: %w", err)
	}

	s.writeOffset = uintptr(newText) - text
	return s.writeOffset, nil
}

//go:build darwin && arm64

package static

import (
	"encoding/binary"
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/pboyd/redefine/internal/cacheflush"
	"github.com/pboyd/redefine/internal/mach"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/sys/unix"
)

var moduledataCopy moduledata

// duplicate clones the program's static data.
//
// If a duplicate already exists that will be returned instead of making another copy.
func (s *Info) duplicate() (*Info, error) {
	if s.dupInfo != nil {
		// Already duplicated, don't try and do it again.
		return s.dupInfo, nil
	}
	if s.isDuplicate() {
		// This is the duplicate, return this instance.
		return s, nil
	}

	ptr, err := unix.MmapPtr(-1, 0, unsafe.Pointer(s.End), s.End-s.Start, syscall.PROT_NONE, unix.MAP_ANON|unix.MAP_PRIVATE|unix.MAP_JIT)
	if err != nil {
		return nil, fmt.Errorf("mmap: %w", err)
	}

	s.offset = uintptr(ptr) - s.Start

	s.dupInfo = &Info{
		offset: s.offset,
		Start:  uintptr(ptr),
		End:    s.End + s.offset,
		datap:  &moduledataCopy,
	}

	// Copy the entire range [text, rodata) into the duplicate mapping.
	// On darwin/arm64 the linker places executable text and read-only
	// sections (go.func.*, pclntab, rodata stubs …) together in the
	// __TEXT segment, so the copy must extend to datap.rodata, not just
	// datap.etext.  Pass datap.rodata as textEnd so that fixADRP
	// correctly leaves ADRPs targeting pages in [etext, rodata)
	// unadjusted — those pages are part of the duplicate.
	err = copyText(
		s.dupInfo.Start+(s.datap.text-s.Start),
		s.datap.text,
		s.datap.rodata-s.Start,
		s.datap.rodata,
	)
	if err != nil {
		s.unduplicate()
		return nil, fmt.Errorf("text: %w", err)
	}

	moduledataCopy = *s.datap
	moduledataCopy.text += s.offset
	moduledataCopy.etext += s.offset
	moduledataCopy.minpc += s.offset
	moduledataCopy.maxpc += s.offset

	moduledataCopy.textsectmap = make([]textsect, len(s.datap.textsectmap))
	for i := range moduledataCopy.textsectmap {
		moduledataCopy.textsectmap[i] = s.datap.textsectmap[i]
		moduledataCopy.textsectmap[i].baseaddr += s.offset
	}

	// Register the duplicate moduledata with the Go runtime so that
	// findfunc and related functions can locate PCs in the duplicate text.
	lastmoduledatap.next = &moduledataCopy

	return s.dupInfo, nil
}

func copyText(destPtr uintptr, srcPtr uintptr, length uintptr, textEnd uintptr) error {
	if length == 0 {
		return nil
	}

	// pageOffset is the address of srcPtr relative to the start of the
	// page. destPtr must have the same value.
	pageOffset := srcPtr &^ pageMask
	if pageOffset != destPtr&^pageMask {
		return errors.New("src and dest have different page offsets")
	}

	// offset is the number of bytes between the destination and source
	offset := destPtr - srcPtr

	srcPages := unsafe.Slice((*byte)(unsafe.Pointer(srcPtr&pageMask)), int((pageOffset+length+pageSize-1)&pageMask))
	destPages := unsafe.Slice((*byte)(unsafe.Pointer(destPtr&pageMask)), int((pageOffset+length+pageSize-1)&pageMask))

	// Make our data RWX and keep it that way forever. Writes are blocked
	// through the Darwin's pthread_jit_write_protect, not mprotect.
	err := unix.Mprotect(destPages, unix.PROT_READ|unix.PROT_WRITE|unix.PROT_EXEC)
	if err != nil {
		return fmt.Errorf("mprotect: %w", err)
	}

	mach.JITWriteUnprotect()
	defer mach.JITWriteProtect()

	src := srcPages[pageOffset : pageOffset+length]
	dest := destPages[pageOffset : pageOffset+length]

	copy(dest, src)

	// Find the duplicate marker in src, then translate that address to dest and set the value to 1.
	*(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(dupMarker())) + offset)) = 1

	if err := fixADRP(dest, srcPtr, textEnd, offset); err != nil {
		return fmt.Errorf("fixADRP: %w", err)
	}

	cacheflush.Flush(destPages)

	return nil
}

// unduplicate frees memory allocated by duplicate.
func (s *Info) unduplicate() error {
	if s.dupInfo == nil {
		return nil
	}

	err := unix.MunmapPtr(unsafe.Pointer(s.dupInfo.Start), s.dupInfo.End-s.dupInfo.Start)
	if err != nil {
		return err
	}

	s.dupInfo = nil

	return nil
}

const (
	// ADR/ADRP is encoded as:
	// --------------------------------------------------
	// | P | lo 2 bits | 10000 | hi 19 bits | 5-bit reg |
	// --------------------------------------------------
	// Mask for the address:
	adrAddressMask = uint32(3<<29 | 0x7ffff<<5)
)

func fixADRP(code []byte, origText, origEtext, offset uintptr) error {
	destBase := uintptr(unsafe.Pointer(unsafe.SliceData(code)))
	srcBase := destBase - offset

	// ADRP always uses 4KB page granularity regardless of OS page size.
	const adrpPageMask = ^uintptr(0xfff)
	origTextPage := origText & adrpPageMask
	origEtextPage := (origEtext + 0xfff) & adrpPageMask

	for i := uintptr(0); i < uintptr(len(code)); i += 4 {
		raw := code[i : i+4]
		inst, err := arm64asm.Decode(raw)
		if err != nil {
			// Just skip bad instructions. It's probably padding or data.
			continue
		}

		destPC := destBase + i
		srcPC := srcBase + i

		switch inst.Op {
		case arm64asm.ADRP:
			oldArg := int64(inst.Args[1].(arm64asm.PCRel))

			// Don't update the address if the target is within the
			// original text. We want those to keep the same relative value
			// so that they'll point to the new text.
			targetPage := uintptr(int64(srcPC&adrpPageMask) + oldArg)
			if targetPage >= origTextPage && targetPage < origEtextPage {
				continue
			}

			newImm := (int64(srcPC&adrpPageMask) + oldArg - int64(destPC&adrpPageMask)) >> 12
			if newImm < -(1<<20) || newImm >= (1<<20) {
				return fmt.Errorf("ADRP at byte offset %d: adjusted immediate %d out of 21-bit signed range", i, newImm)
			}
			newArg := uint32(newImm)

			encoded := binary.LittleEndian.Uint32(raw) &^ adrAddressMask
			encoded |= (newArg & 3) << 29             // Lowest 2 bits to bits 30 and 29
			encoded |= ((newArg >> 2) & 0x7ffff) << 5 // Highest 19 bits to bits 23 to 5
			binary.LittleEndian.PutUint32(raw, encoded)

		}
	}
	return nil
}

func patchRodataCodePtrs(offset uintptr, moddata *moduledata) error {
	if moddata.etext >= moddata.noptrdata {
		return nil
	}

	mapStart := (moddata.etext + pageSize - 1) & pageMask
	mapEnd := moddata.noptrdata & pageMask
	if mapStart >= mapEnd {
		return nil
	}

	entries := make(map[uintptr]struct{}, len(moddata.ftab))
	for _, ft := range moddata.ftab {
		entries[moddata.text+uintptr(ft.entryoff)] = struct{}{}
	}

	size := mapEnd - mapStart

	tmpPtr, err := unix.MmapPtr(-1, 0, nil, size,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_PRIVATE|unix.MAP_ANON)
	if err != nil {
		return fmt.Errorf("mmap temp rodata (%d bytes): %w", size, err)
	}
	tmpSlice := unsafe.Slice((*byte)(tmpPtr), int(size))
	copy(tmpSlice, unsafe.Slice((*byte)(unsafe.Pointer(mapStart)), int(size)))

	// ignore pclntable area, because patching those pointers caused crashes.
	pclnStart := uintptr(unsafe.Pointer(unsafe.SliceData(moddata.pclntable)))
	pclnEnd := pclnStart + uintptr(len(moddata.pclntable))

	for addr := mapStart; addr+8 <= mapEnd; addr += 8 {
		if addr >= pclnStart && addr < pclnEnd {
			continue
		}

		off := addr - mapStart
		val := *(*uintptr)(unsafe.Pointer(&tmpSlice[off]))
		if val >= moddata.text && val < moddata.etext {
			if _, ok := entries[val]; ok {
				*(*uintptr)(unsafe.Pointer(&tmpSlice[off])) = val + offset
			}
		}
	}

	_, err = mach.VmRemap(mapStart, uintptr(tmpPtr), size)
	if err != nil {
		unix.MunmapPtr(tmpPtr, size)
		return fmt.Errorf("vm_remap rodata (%d bytes at %#x): %w", size, mapStart, err)
	}

	if err := unix.Mprotect(unsafe.Slice((*byte)(unsafe.Pointer(mapStart)), int(size)), unix.PROT_READ); err != nil {
		return fmt.Errorf("mprotect rodata to r: %w", err)
	}

	unix.MunmapPtr(tmpPtr, size)

	return nil
}

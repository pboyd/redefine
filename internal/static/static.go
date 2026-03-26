package static

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

//go:linkname findfunc runtime.findfunc
func findfunc(pc uintptr) funcInfo

//go:linkname lastmoduledatap runtime.lastmoduledatap
var lastmoduledatap *moduledata

var pageSize = uintptr(syscall.Getpagesize())
var pageMask = ^(pageSize - 1)

var info Info
var infoInit sync.Once

func GetInfo() *Info {
	infoInit.Do(func() {
		// The info is based on the module of this function. This is
		// typically the main program, but it may not be under
		// -buildmode=plugin or -buildmode=shared.
		pc, _, _, _ := runtime.Caller(0)
		datap := findfunc(pc).datap

		// Align start and end to page boundaries
		start := datap.text & pageMask
		length := ((datap.end - start) + pageSize - 1) & pageMask
		end := start + length

		info = Info{
			datap: datap,
			Start: datap.text & pageMask,
			End:   end,
		}
	})

	return &info
}

type Info struct {
	// Delta from an original address to the duplicate.
	writeOffset uintptr

	Start, End uintptr

	datap *moduledata
}

// Text returns the address of the beginning and end of the text segment.
func (s *Info) Text() (text uintptr, etext uintptr) {
	text = s.datap.text
	etext = s.datap.etext
	return
}

// FuncSlice returns a slice containing the machine instructions for a function.
//
// The returned uintptr is the address that the function executes from, which
// differs from the slice data address on Darwin.
func (s *Info) FuncSlice(fn any) ([]byte, uintptr, error) {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return nil, 0, fmt.Errorf("not a function, kind: %v", fnv.Kind())
	}
	entry := fnv.Pointer()

	datap := findfunc(entry).datap
	if datap == nil {
		return nil, 0, errors.New("no moduledata for function")
	}

	text := datap.text
	etext := datap.etext
	ftab := datap.ftab

	// To find the length, look at the offsets of every function and find
	// the one that comes immediately after this one.

	// TODO: ftab seems to be ordered, can we rely on that to speed this up?

	funcOffset := uint32(entry - text)
	length := uint32(etext - entry)

	for _, ft := range ftab {
		// Does this function come before the one we're looking for?
		if ft.entryoff <= funcOffset {
			continue
		}

		// Is the distance between these two functions less than what we've seen before?
		testLength := ft.entryoff - funcOffset
		if testLength < length {
			length = testLength
		}
	}

	// If there's a writable version, use that instead of the executable one for slice.
	writeOffset, err := s.getWriteOffset()
	if err != nil {
		return nil, 0, fmt.Errorf("unable to get writeOffset: %w", err)
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(entry+writeOffset)), length), entry, nil
}

func (s *Info) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "addr: 0x%x\n", uintptr(unsafe.Pointer(s)))
	fmt.Fprintf(&b, "writeOffset: 0x%x\n", s.writeOffset)
	fmt.Fprintf(&b, "Start: 0x%x\n", s.Start)
	fmt.Fprintf(&b, "End: 0x%x\n", s.End)
	fmt.Fprintf(&b, "datap:\n")
	fmt.Fprintf(&b, "  %-10s 0x%x - 0x%x\n", "text", s.datap.text, s.datap.etext)
	fmt.Fprintf(&b, "  %-10s 0x%x - 0x%x\n", "noptrdata", s.datap.noptrdata, s.datap.enoptrdata)
	fmt.Fprintf(&b, "  %-10s 0x%x - 0x%x\n", "data", s.datap.data, s.datap.edata)
	fmt.Fprintf(&b, "  %-10s 0x%x - 0x%x\n", "bss", s.datap.bss, s.datap.ebss)
	fmt.Fprintf(&b, "  %-10s 0x%x - 0x%x\n", "noptrbss", s.datap.noptrbss, s.datap.enoptrbss)
	fmt.Fprintf(&b, "  %-10s 0x%x - 0x%x\n", "covctrs", s.datap.covctrs, s.datap.ecovctrs)
	fmt.Fprintf(&b, "  %-10s 0x%x - 0x%x\n", "types", s.datap.types, s.datap.etypes)
	fmt.Fprintf(&b, "  %-10s 0x%x\n", "end", s.datap.end)
	fmt.Fprintf(&b, "  %-10s 0x%x\n", "gcdata", s.datap.gcdata)
	fmt.Fprintf(&b, "  %-10s 0x%x\n", "gcbss", s.datap.gcbss)

	return b.String()
}

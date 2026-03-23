//go:build darwin && arm64

package static

import (
	"fmt"
	"runtime"
)

func Fork() error {
	if runningInDuplicate() {
		// Nothing to do
		return nil
	}

	info := GetInfo()
	_, err := info.duplicate()
	if err != nil {
		return err
	}

	err = patchRodataCodePtrs(info.offset, info.datap)
	if err != nil {
		return fmt.Errorf("patchRodataCodePtrs: %w", err)
	}

	origText, origEtext := info.originalText()

	for f := getFrame(); f != nil; f = f.next {
		if f.lr >= origText && f.lr < origEtext {
			f.lr += info.offset
		}
	}

	return nil
}

type frame struct {
	// By convention, Go stores the address of the next frame followed by
	// the return address.
	next *frame
	lr   uintptr
}

func (f *frame) Func() *runtime.Func {
	return runtime.FuncForPC(f.lr)
}

func getFrame() *frame

type g struct {
	stack stack
}

type stack struct {
	lo uintptr
	hi uintptr
}

func getg() *g

func dupMarker() *uint32

func runningInDuplicate() bool {
	return *dupMarker() != 0
}

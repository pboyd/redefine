//go:build darwin && arm64

package redefine

import (
	"github.com/pboyd/redefine/internal/mach"
	"golang.org/x/sys/unix"
)

func mprotectHook(inner func(int) error) func(int) error {
	return func(prot int) error {
		if prot&unix.PROT_WRITE != 0 {
			mach.JITWriteUnprotect()
		} else {
			mach.JITWriteProtect()
		}
		return nil
	}
}

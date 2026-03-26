//go:build darwin && arm64

package redefine

import (
	"golang.org/x/sys/unix"
)

const (
	mprotectExec = unix.PROT_EXEC
	mprotectRX   = unix.PROT_READ | unix.PROT_EXEC
	mprotectRWX  = unix.PROT_READ | unix.PROT_WRITE
)

// makeRWX and makeRX are no-ops on darwin/arm64.
func makeRWX(buf []byte) error { return nil }
func makeRX(buf []byte) error  { return nil }

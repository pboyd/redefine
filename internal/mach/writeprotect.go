//go:build darwin && arm64

package mach

import (
	"runtime"
)

/*
#include <pthread.h>
*/
import "C"

// JITWriteUnprotect is a wrapper around pthread_jit_write_protect_np to allow
// writes to MAP_JIT allocated data sections.
//
// Because the setting is thread-specific, the goroutine will be locked to its
// OS thread until JITWriteUnprotect is called.
func JITWriteUnprotect() {
	runtime.LockOSThread()
	C.pthread_jit_write_protect_np(0)
}

// JITWriteProtect reverses the effects of JITWriteUnprotect.
func JITWriteProtect() {
	C.pthread_jit_write_protect_np(1)
	runtime.UnlockOSThread()
}

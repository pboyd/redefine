//go:build darwin && arm64

package redefine

/*
#include <pthread.h>
*/
import "C"
import (
	"runtime"

	"golang.org/x/sys/unix"
)

func mprotectHook(inner func(int) error) func(int) error {
	return func(prot int) error {
		// Instead of calling mprotect, just use Darwin's
		// pthread_jit_write_protect_np which is effectively the same
		// in this case.

		// This value is thread specific, so lock the running goroutine
		// to the system thread. This assumes that this function is
		// called in BeginMutate/EndMutate pairs.

		if prot&unix.PROT_WRITE != 0 {
			runtime.LockOSThread()
			C.pthread_jit_write_protect_np(0)
		} else {
			C.pthread_jit_write_protect_np(1)
			runtime.UnlockOSThread()
		}
		return nil
	}
}

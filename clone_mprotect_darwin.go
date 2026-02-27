//go:build darwin && arm64

package redefine

import "golang.org/x/sys/unix"

/*
	#include <pthread.h>
*/
import "C"

func mprotectHook(inner func(int) error) func(int) error {
	return func(prot int) error {
		if prot&unix.PROT_WRITE != 0 {
			C.pthread_jit_write_protect_np(0)
		} else {
			C.pthread_jit_write_protect_np(1)
		}

		err := inner(prot)

		// Restore write protection after an error
		if err != nil && prot&unix.PROT_WRITE != 0 {
			C.pthread_jit_write_protect_np(1)
		}

		return err
	}
}

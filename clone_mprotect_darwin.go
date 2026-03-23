//go:build darwin && arm64

package redefine

/*
#include <pthread.h>
#include <string.h>

// jit_memcpy copies len bytes from src to dst within a JIT write-protected
// scope: pthread_jit_write_protect_np(0) before and (1) after, with an
// I-cache flush in between. The entire operation runs in C so that the return
// to Go code happens after MAP_JIT pages are back in execute mode.
static void jit_memcpy(void *dst, const void *src, size_t len) {
	pthread_jit_write_protect_np(0);
	memcpy(dst, src, len);
	__builtin___clear_cache(dst, (char *)dst + len);
	pthread_jit_write_protect_np(1);
}
*/
import "C"
import "unsafe"

func mprotectHook(inner func(int) error) func(int) error {
	return inner
}

// writeJITCode copies src into dst on MAP_JIT pages.  The JIT write-protect
// toggle and I-cache flush happen entirely in C, so the return to Go (which
// executes from the duplicate MAP_JIT text) is always in execute mode.
func writeJITCode(dst, src []byte) {
	C.jit_memcpy(
		unsafe.Pointer(unsafe.SliceData(dst)),
		unsafe.Pointer(unsafe.SliceData(src)),
		C.size_t(len(src)),
	)
}

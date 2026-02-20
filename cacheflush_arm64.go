//go:build arm64

package redefine

import "unsafe"

/*
static void cacheflush(char *start, char *end) {
	__builtin___clear_cache(start, end);
}
*/
import "C"

func cacheflush(buf []byte) {
	start := unsafe.Pointer(unsafe.SliceData(buf))
	end := unsafe.Pointer(uintptr(len(buf)) + uintptr(start))
	C.cacheflush((*C.char)(start), (*C.char)(end))
}

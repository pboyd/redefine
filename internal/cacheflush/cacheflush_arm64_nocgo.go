//go:build arm64 && !cgo

package cacheflush

// arm64 requires a C compiler to flush the instruction cache.
// Install a C compiler and build with CGO_ENABLED=1.
func Flush(buf []byte) {
	arm64_requires_cgo_for_instruction_cache_flushing()
}

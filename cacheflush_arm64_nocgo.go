//go:build arm64 && !cgo

package redefine

// arm64 requires a C compiler to flush the instruction cache.
// Install a C compiler and build with CGO_ENABLED=1.
func cacheflush(buf []byte) {
	arm64_requires_cgo_for_instruction_cache_flushing()
}

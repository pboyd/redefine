//go:build !arm64

package redefine

// This isn't needed on amd64. The arm64 version uses the C builtin which is a
// no-op, but avoiding cgo makes cross-compiling easier.
func cacheflush(buf []byte) {}

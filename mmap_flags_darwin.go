//go:build darwin && arm64

package redefine

import "golang.org/x/sys/unix"

// Darwin has no equivalent to MAP_FIXED_NOREPLACE. But MAP_JIT is required to
// use PROT_WRITE and PROT_EXEC together.
const _MMAP_FLAGS = unix.MAP_JIT

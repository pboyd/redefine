//go:build darwin && arm64

package redefine

import "golang.org/x/sys/unix"

// Darwin has no equivalent to MAP_FIXED_NOREPLACE. Use MAP_JIT so we can swap
// between rw- and r-x.
const _MMAP_FLAGS = unix.MAP_JIT

//go:build linux

package redefine

import "golang.org/x/sys/unix"

const _MMAP_FLAGS = unix.MAP_FIXED_NOREPLACE

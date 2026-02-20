//go:build freebsd

package redefine

import "golang.org/x/sys/unix"

// MAP_FIXED with MAP_EXCL seems mostly equivalent to MAP_FIXED_NOREPLACE on
// Linux
//
// https://man.freebsd.org/cgi/man.cgi?mmap(2)
const _MAP_FIXED_NOREPLACE = unix.MAP_FIXED | unix.MAP_EXCL

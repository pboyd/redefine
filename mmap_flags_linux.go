//go:build linux

package redefine

import "golang.org/x/sys/unix"

const _MAP_FIXED_NOREPLACE = unix.MAP_FIXED_NOREPLACE

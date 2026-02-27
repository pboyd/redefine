//go:build netbsd || openbsd || windows || (darwin && amd64)

package redefine

// Darwin, NetBSD, OpenBSD and Windows don't have an equivalent to
// MAP_FIXED_NOREPLACE.
//
// On BSD, MAP_FIXED would almost work except that it would replace existing
// mappings. We'll search for a suitable address.
//
// Darwin/arm64 is handled elsewhere because it needs special behavior. But
// Darwin/amd64 is more like its BSD roots.
//
// https://man.netbsd.org/mmap.2
// https://man.openbsd.org/mmap.2
// https://learn.microsoft.com/en-us/windows/win32/api/memoryapi/nf-memoryapi-virtualalloc
const _MMAP_FLAGS = 0

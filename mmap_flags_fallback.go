//go:build darwin || netbsd || openbsd || windows

package redefine

// Darwin, NetBSD, OpenBSD and Windows don't have an equivalent to
// MAP_FIXED_NOREPLACE. On BSD, MAP_FIXED would almost work except that it
// would replace existing mappings. We'll have to trust the OS to give us a
// suitable address based on our request.
//
// https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/mmap.2.html
// https://man.netbsd.org/mmap.2
// https://man.openbsd.org/mmap.2
// https://learn.microsoft.com/en-us/windows/win32/api/memoryapi/nf-memoryapi-virtualalloc
const _MAP_FIXED_NOREPLACE = 0

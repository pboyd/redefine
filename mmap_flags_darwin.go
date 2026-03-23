//go:build darwin && arm64

package redefine

// The allocator uses non-MAP_JIT pages so that pthread_jit_write_protect_np
// is never toggled from MAP_JIT (duplicate text) code. W^X is maintained by
// using PROT_READ|PROT_WRITE during code generation and PROT_READ|PROT_EXEC
// during execution, toggled via regular mprotect.
const _MMAP_FLAGS = 0

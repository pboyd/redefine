// Redefine Go functions at runtime
//
// This exists because I wondered if it were possible to rewrite a Go function
// like some interpreted languages allow (Ruby being a prominent example). This
// is a fun experiment, but do not use it for production code.
//
// This project is fundamentally non-portable. OS/Arch support:
//   - Full support: Linux/amd64, Windows/amd64, Darwin/amd64, Linux/arm64
//   - Might work (untested, but it compiles): FreeBSD/amd64, OpenBSD/amd64, NetBSD/amd64
//   - Also might work: Windows/arm64 (I lack a working build environment)
//   - Known broken: Darwin/arm64 (EACCES errors from mprotect)
//
// Other limitations:
//   - Relies on internal Go APIs that can break at any time
//   - Silently fails to redefine inline and generic functions
package redefine

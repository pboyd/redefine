// Redefine Go functions at runtime
//
// This exists because I wondered if it were possible to rewrite a Go function
// like some interpreted languages allow (Ruby being a prominent example). This
// is a fun experiment, but do not use it for production code.
//
// This project is fundamentally non-portable. OS/Arch support:
//   - Full support: Linux, Windows, Darwin/MacOS on amd64 and arm64
//   - Might work (untested, but it compiles): FreeBSD, OpenBSD, NetBSD on amd64
//
// Other limitations:
//   - Relies on internal Go APIs that can break at any time
//   - Silently fails to redefine inline and generic functions
package redefine

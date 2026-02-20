// Redefine Go functions at runtime
//
// This exists because I wondered if it were possible to rewrite a Go function
// like some interpreted languages allow (Ruby being a prominent example). This
// is a fun experiment, but do not use it for production code.
//
// Known limitations:
//   - Only supports amd64
//   - Compiles on FreeBSD, OpenBSD and NetBSD, but these are untested
//   - Relies on internal Go APIs that can break at any time
//   - Silently fails to redefine inline and generic functions
package redefine

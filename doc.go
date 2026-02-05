// Redefine Go functions at runtime
//
// I wondered if it was possible to rewrite a Go function like some interpreted
// languages allow (Ruby being a prominent example). It turns out to be
// possible and this package is the proof-of-concept. You shouldn't use this.
//
// Limitations:
//   - Only supports amd64 on Unix or Linux
//   - Relies on internal Go APIs that can break at any time
//   - Silently fails to redefine inline functions
//   - Silently fails to redefine generic functions
//   - Probably some bugs I don't know about.
package redefine

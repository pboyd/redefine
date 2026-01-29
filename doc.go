// An experimental attempt to redefine Go functions at runtime.
//
// I wondered if it was possible to rewrite a Go function like some interpreted
// languages allow (Ruby being a prominent example). It turns out to be
// possible and this package is the proof-of-concept. You shouldn't use this.
//
// Currently limitations:
//   - Only compiles on linux/amd64
//   - Once redefined, the original function is lost
//   - Relies on internal Go APIs that can break at any time
//   - Silently fails to redefine inline functions
//   - Probably some bugs I don't know about.
//
// Did I mention you shouldn't use this?
package redefine

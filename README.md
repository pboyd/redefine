## redefine

[![Go Reference](https://pkg.go.dev/badge/github.com/pboyd/redefine.svg)](https://pkg.go.dev/github.com/pboyd/redefine)

Highly experimental package to redefine Go functions at runtime as some interpreted languages allow (Ruby, Perl, etc.). I wrote about how this works and some of the limitations [here](https://pboyd.io/posts/redefining-go-functions/). This is a fun experiment, but do not use it for production code.

```go
package main

import (
	"fmt"
	"time"

	"github.com/pboyd/redefine"
)

func main() {
	redefine.Func(time.Now, func() time.Time {
		return time.Date(2000, 1, 1, 17, 0, 0, 0, time.FixedZone("somewhere", -5))
	})

	fmt.Printf("It's %s\n", time.Now().Format("3:04 PM MST"))
}
```

Outputs:

```
It's 5:00 PM somewhere
```

## Compatibility

| OS | Architecture | Status | Notes |
|----|-------------|--------|-------|
| Linux | amd64 | Full | |
| Windows | amd64 | Full | |
| Darwin (macOS) | amd64 | Full | |
| Linux | arm64 | Full | |
| Windows | arm64 | Full | |
| FreeBSD | amd64 | Untested | Compiles but untested |
| OpenBSD | amd64 | Untested | Compiles but untested |
| NetBSD | amd64 | Untested | Compiles but untested |
| Darwin (macOS) | arm64 | Broken | `mprotect` returns EACCES |

## FAQ

### Yikes! Why?

I wondered if it were possible, and it was. Had I searched online first, I would have found [github.com/bouk/monkey](https://github.com/bouk/monkey), which existed 11 years prior.

### Can I use this to mock test functions?

Yes, but real implementations are almost always better than mocks. If you really must have a mock, use a package meant for that ([github.com/stretchr/testify/mock](https://pkg.go.dev/github.com/stretchr/testify/mock) or [github.com/uber-go/mock](https://pkg.go.dev/github.com/uber-go/mock) to name a couple). If you want to mock time in particular, try [syntest](https://pkg.go.dev/testing/synctest#hdr-Time).

Or just use a function pointer in your code:

```Go
package main

import (
	"fmt"
	"math/rand"
)

var myRandIntn = rand.Intn

func main() {
	fmt.Printf("Your lucky number is: %d\n", myRandIntn(100))
}
```

And swap it out for a test:

```Go
func TestFoo(t *testing.T) {
	myRandIntn = func(n int) int {
		return 42
	}
	// ...
}
```

### Can I use this to patch broken and unmaintained dependencies?

Yes, but anything you can do with this package you can accomplish with conventional function wrapping. Of course, if the dependency is truly unmaintained and you have a patch, why not make the world a better place and maintain a fork?

### Can I use this to log when a function is called?

Yes, but again, you can do that by wrapping your calls to the function in the normal way.

### Can I use this to gather timing information about functions in a dependency?

Yes, but [pprof](https://pkg.go.dev/runtime/pprof) does this better. Note that [`redefine.Original`](https://pkg.go.dev/github.com/pboyd/redefine#Original) adds some overhead.

### Can I use this for...

Maybe, but I urge you not to. This is a toy, not a production-ready tool. At best, your code will be harder to reason about. At worst, you have bugs that crash your program far from the problem's source with unhelpful stack traces. In between those extremes, you have non-portable code. Why do that to yourself?

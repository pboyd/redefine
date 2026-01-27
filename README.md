## redefine

Highly experimental package to redefine Go functions at runtime.

If you think you need this, you're almost certainly doing something wrong. It exists because I wondered if were possible to redefine Go functions like some interpreted allow (Ruby, Perl, etc.).

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

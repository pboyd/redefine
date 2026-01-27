package redefine_test

import (
	"fmt"
	"time"

	"github.com/pboyd/redefine"
)

func ExampleFunc() {
	redefine.Func(time.Now, func() time.Time {
		return time.Date(2000, 1, 1, 17, 0, 0, 0, time.FixedZone("somewhere", -5))
	})

	fmt.Printf("It's %s\n", time.Now().Format("3:04 PM MST"))

	// Output: It's 5:00 PM somewhere
}

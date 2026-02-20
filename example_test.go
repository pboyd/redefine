package redefine_test

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pboyd/redefine"
)

func ExampleFunc() {
	redefine.Func(time.Now, func() time.Time {
		return time.Date(2000, 1, 1, 17, 0, 0, 0, time.FixedZone("somewhere", -5))
	})
	defer redefine.Restore(time.Now)

	fmt.Printf("It's %s\n", time.Now().Format("3:04 PM MST"))
	// Output: It's 5:00 PM somewhere
}

type myResolver net.Resolver

func (*myResolver) LookupHost(context.Context, string) ([]string, error) {
	return []string{"127.0.0.1"}, nil
}

func ExampleMethod() {
	redefine.Method((*net.Resolver).LookupHost, (*myResolver).LookupHost)
	defer redefine.Restore((*net.Resolver).LookupHost)

	addrs, _ := net.DefaultResolver.LookupHost(context.Background(), "www.google.com")
	fmt.Printf("www.google.com has addresses %v", addrs)
	// Output: www.google.com has addresses [127.0.0.1]
}

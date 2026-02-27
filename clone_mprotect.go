//go:build !(darwin && arm64)

package redefine

func mprotectHook(inner func(int) error) func(int) error {
	return inner
}

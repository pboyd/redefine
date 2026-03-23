//go:build !darwin || !arm64

package static

func Fork() error {
	return nil
}

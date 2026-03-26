//go:build !darwin || !arm64

package static

func (s *Info) getWriteOffset() (uintptr, error) {
	return 0, nil
}

//go:build !amd64

package redefine

func _cloneFunc[T any](fn T, originalCode []byte) (*clonedFunc[T], error) {
	return &clonedFunc[T]{}, nil
}

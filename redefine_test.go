package redefine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:noinline
func a() string {
	return "a"
}

func b() string {
	return "b"
}

func TestFunc(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("a", a())
	assert.NoError(Func(a, b))
	assert.Equal("b", a())
}

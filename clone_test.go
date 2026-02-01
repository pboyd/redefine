package redefine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func cloneFunc(v uint8) uint16 {
	return uint16(v)<<8 | uint16(v)
}

func TestClone(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(uint16(0x0f0f), cloneFunc(0xf))

	cf, err := CloneFunc(cloneFunc)
	if assert.NoError(err) && assert.NotNil(cf) {
		assert.Equal(uint16(0x0f0f), cf.Func(0xf))

		// Verify that the returned function exists in the
		// cloneAllocator's memory
		assert.True(cloneAllocator.Contains(cf.Func))

		cf.Free()
	}
}

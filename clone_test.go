package redefine

import (
	"encoding/hex"
	"hash/fnv"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func simpleTestCloneFunc(v uint8) uint16 {
	return uint16(v)<<8 | uint16(v)
}

func testCloneFuncWithLotsOfCalls(v int) string {
	// Just need to make a lot of function calls
	h := fnv.New32()
	io.WriteString(h, strconv.Itoa(v))
	buf := h.Sum(nil)
	return hex.EncodeToString(buf)
}

func testCloneFuncWithData() string {
	// FIXME: Use another value.
	return ""
}

func TestClone(t *testing.T) {
	t.Run("clone a simple function", func(t *testing.T) {
		assert := assert.New(t)

		result := simpleTestCloneFunc(0xf)
		cf, err := CloneFunc(simpleTestCloneFunc)
		if assert.NoError(err) && assert.NotNil(cf) {
			t.Cleanup(cf.Free)
			assert.Equal(result, cf.Func(0xf))

			// Verify that the returned function exists in the
			// cloneAllocator's memory
			assert.True(cloneAllocator.Contains(cf.Func))
		}
	})

	t.Run("clone a function with hard-coded data", func(t *testing.T) {
		assert := assert.New(t)

		result := testCloneFuncWithData()
		cf, err := CloneFunc(testCloneFuncWithData)
		if assert.NoError(err) && assert.NotNil(cf) {
			t.Cleanup(cf.Free)
			assert.Equal(result, cf.Func())
		}
	})

	/*
		// FIXME: Panics hard right now
		t.Run("clone a function with CALL instructions", func(t *testing.T) {
			assert := assert.New(t)

			result := testCloneFuncWithLotsOfCalls(25)
			cf, err := CloneFunc(testCloneFuncWithLotsOfCalls)
			if assert.NoError(err) && assert.NotNil(cf) {
				t.Cleanup(cf.Free)
				assert.Equal(result, cf.Func(25))

				// Verify that the returned function exists in the
				// cloneAllocator's memory
				assert.True(cloneAllocator.Contains(cf.Func))
			}
		})
	*/
}

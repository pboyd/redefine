package redefine

import (
	"encoding/hex"
	"hash/fnv"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testCloneFuncWithLotsOfCalls(v int) string {
	// Just need to make a lot of function calls
	h := fnv.New32()
	io.WriteString(h, strconv.Itoa(v))
	buf := h.Sum(nil)
	return hex.EncodeToString(buf)
}

func TestClone(t *testing.T) {
	assert := assert.New(t)

	result := testCloneFuncWithLotsOfCalls(25)
	cf, err := CloneFunc(testCloneFuncWithLotsOfCalls)
	if assert.NoError(err) && assert.NotNil(cf) {
		t.Cleanup(cf.Free)
		assert.Equal(result, cf.Func(25))

		assert.True(cloneAllocator.Contains(cf.Func))
	}
}

func simpleTestCloneFunc(v uint8) uint16 {
	return uint16(v)<<8 | uint16(v)
}

func testCloneFuncWithOneCall(v int) string {
	return strconv.Itoa(v + 1)
}

func testCloneFuncWithData() string {
	return "something static"
}

func TestClone_VariousFunctions(t *testing.T) {
	cases := map[string]struct {
		call         func() any
		cloneAndCall func(t *testing.T) (any, error)
	}{
		"simple function": {
			call: func() any {
				return simpleTestCloneFunc(0xf)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(simpleTestCloneFunc)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(0xf), nil
			},
		},
		"function with static data": {
			call: func() any {
				return testCloneFuncWithData()
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncWithData)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(), nil
			},
		},
		"time.Now": {
			call: func() any {
				return time.Now().Truncate(time.Hour)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(time.Now)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func().Truncate(time.Hour), nil
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			actual, err := tc.cloneAndCall(t)
			if assert.NoError(err) {
				assert.Equal(tc.call(), actual)
			}
		})
	}
}

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

func testCloneFuncMultipleParams(a, b int) int {
	return a + b
}

func testCloneFuncMultipleReturns(v int) (int, error) {
	if v < 0 {
		return 0, io.ErrUnexpectedEOF
	}
	return v * 2, nil
}

func testCloneFuncFloat(f float64) float64 {
	return f * 3.14159
}

func testCloneFuncWithLoop(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		sum += i
	}
	return sum
}

func testCloneFuncWithConditional(v int) string {
	if v > 100 {
		return "large"
	} else if v > 10 {
		return "medium"
	} else {
		return "small"
	}
}

func testCloneFuncWithSlice(vals []int) int {
	sum := 0
	for _, v := range vals {
		sum += v
	}
	return sum
}

func testCloneFuncWithPointer(p *int) int {
	if p == nil {
		return 0
	}
	return *p * 10
}

func testCloneFuncVariadic(vals ...int) int {
	sum := 0
	for _, v := range vals {
		sum += v
	}
	return sum
}

func testCloneFuncGeneric[T int | float64](v T) T {
	return v + v
}

func testCloneFuncComplex128(c complex128) complex128 {
	return c * complex(2, 3)
}

func testCloneFuncInt64(a int64, b int64) int64 {
	return a << 32 | b
}

func testCloneFuncNested(x int) int {
	helper := func(y int) int {
		return y * 2
	}
	return helper(x) + 1
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
		"function with one call": {
			call: func() any {
				return testCloneFuncWithOneCall(42)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncWithOneCall)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(42), nil
			},
		},
		"multiple parameters": {
			call: func() any {
				return testCloneFuncMultipleParams(10, 32)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncMultipleParams)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(10, 32), nil
			},
		},
		"multiple returns success": {
			call: func() any {
				v, e := testCloneFuncMultipleReturns(25)
				return struct {
					val int
					err error
				}{v, e}
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncMultipleReturns)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				v, e := cf.Func(25)
				return struct {
					val int
					err error
				}{v, e}, nil
			},
		},
		"multiple returns error": {
			call: func() any {
				v, e := testCloneFuncMultipleReturns(-1)
				return struct {
					val int
					err error
				}{v, e}
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncMultipleReturns)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				v, e := cf.Func(-1)
				return struct {
					val int
					err error
				}{v, e}, nil
			},
		},
		"float64 operations": {
			call: func() any {
				return testCloneFuncFloat(2.5)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncFloat)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(2.5), nil
			},
		},
		"function with loop": {
			call: func() any {
				return testCloneFuncWithLoop(10)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncWithLoop)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(10), nil
			},
		},
		"function with conditional small": {
			call: func() any {
				return testCloneFuncWithConditional(5)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncWithConditional)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(5), nil
			},
		},
		"function with conditional medium": {
			call: func() any {
				return testCloneFuncWithConditional(50)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncWithConditional)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(50), nil
			},
		},
		"function with conditional large": {
			call: func() any {
				return testCloneFuncWithConditional(500)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncWithConditional)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(500), nil
			},
		},
		"function with slice": {
			call: func() any {
				return testCloneFuncWithSlice([]int{1, 2, 3, 4, 5})
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncWithSlice)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func([]int{1, 2, 3, 4, 5}), nil
			},
		},
		"function with nil pointer": {
			call: func() any {
				return testCloneFuncWithPointer(nil)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncWithPointer)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(nil), nil
			},
		},
		"function with non-nil pointer": {
			call: func() any {
				v := 7
				return testCloneFuncWithPointer(&v)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncWithPointer)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				v := 7
				return cf.Func(&v), nil
			},
		},
		"variadic function empty": {
			call: func() any {
				return testCloneFuncVariadic()
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncVariadic)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(), nil
			},
		},
		"variadic function single": {
			call: func() any {
				return testCloneFuncVariadic(42)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncVariadic)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(42), nil
			},
		},
		"variadic function multiple": {
			call: func() any {
				return testCloneFuncVariadic(1, 2, 3, 4, 5)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncVariadic)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(1, 2, 3, 4, 5), nil
			},
		},
		"generic function int": {
			call: func() any {
				return testCloneFuncGeneric(21)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncGeneric[int])
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(21), nil
			},
		},
		"generic function float64": {
			call: func() any {
				return testCloneFuncGeneric(3.14)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncGeneric[float64])
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(3.14), nil
			},
		},
		"complex128 operations": {
			call: func() any {
				return testCloneFuncComplex128(complex(1, 2))
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncComplex128)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(complex(1, 2)), nil
			},
		},
		"int64 operations": {
			call: func() any {
				return testCloneFuncInt64(0x12345678, 0xABCDEF00)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncInt64)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(0x12345678, 0xABCDEF00), nil
			},
		},
		"function with closure": {
			call: func() any {
				return testCloneFuncNested(5)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := CloneFunc(testCloneFuncNested)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(5), nil
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

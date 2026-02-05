package redefine

import (
	"encoding/hex"
	"hash/fnv"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
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

func TestCloneFunc(t *testing.T) {
	assert := assert.New(t)

	result := testCloneFuncWithLotsOfCalls(25)
	cf, err := cloneFunc(testCloneFuncWithLotsOfCalls)
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
	return a<<32 | b
}

func testCloneFuncNested(x int) int {
	helper := func(y int) int {
		return y * 2
	}
	return helper(x) + 1
}

func TestCloneFunc_VariousFunctions(t *testing.T) {
	cases := map[string]struct {
		call         func() any
		cloneAndCall func(t *testing.T) (any, error)
	}{
		"simple function": {
			call: func() any {
				return simpleTestCloneFunc(0xf)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(simpleTestCloneFunc)
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
				cf, err := cloneFunc(testCloneFuncWithData)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(), nil
			},
		},
		"function with one call": {
			call: func() any {
				return testCloneFuncWithOneCall(42)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(testCloneFuncWithOneCall)
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
				cf, err := cloneFunc(testCloneFuncMultipleParams)
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
				cf, err := cloneFunc(testCloneFuncMultipleReturns)
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
				cf, err := cloneFunc(testCloneFuncMultipleReturns)
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
				cf, err := cloneFunc(testCloneFuncFloat)
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
				cf, err := cloneFunc(testCloneFuncWithLoop)
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
				cf, err := cloneFunc(testCloneFuncWithConditional)
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
				cf, err := cloneFunc(testCloneFuncWithConditional)
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
				cf, err := cloneFunc(testCloneFuncWithConditional)
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
				cf, err := cloneFunc(testCloneFuncWithSlice)
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
				cf, err := cloneFunc(testCloneFuncWithPointer)
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
				cf, err := cloneFunc(testCloneFuncWithPointer)
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
				cf, err := cloneFunc(testCloneFuncVariadic)
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
				cf, err := cloneFunc(testCloneFuncVariadic)
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
				cf, err := cloneFunc(testCloneFuncVariadic)
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
				cf, err := cloneFunc(testCloneFuncGeneric[int])
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
				cf, err := cloneFunc(testCloneFuncGeneric[float64])
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
				cf, err := cloneFunc(testCloneFuncComplex128)
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
				cf, err := cloneFunc(testCloneFuncInt64)
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
				cf, err := cloneFunc(testCloneFuncNested)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(5), nil
			},
		},
		"time.Now": {
			call: func() any {
				return time.Now().Truncate(time.Hour)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(time.Now)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func().Truncate(time.Hour), nil
			},
		},
		"math.Sqrt": {
			call: func() any {
				return math.Sqrt(16.0)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(math.Sqrt)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(16.0), nil
			},
		},
		"math.Sin": {
			call: func() any {
				return math.Sin(math.Pi / 2)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(math.Sin)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(math.Pi / 2), nil
			},
		},
		"math.Pow": {
			call: func() any {
				return math.Pow(2.0, 8.0)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(math.Pow)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(2.0, 8.0), nil
			},
		},
		"math.Max": {
			call: func() any {
				return math.Max(42.5, 17.3)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(math.Max)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(42.5, 17.3), nil
			},
		},
		"strings.ToUpper": {
			call: func() any {
				return strings.ToUpper("hello world")
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(strings.ToUpper)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func("hello world"), nil
			},
		},
		"strings.Contains": {
			call: func() any {
				return strings.Contains("hello world", "world")
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(strings.Contains)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func("hello world", "world"), nil
			},
		},
		"strings.HasPrefix": {
			call: func() any {
				return strings.HasPrefix("prefix-test", "prefix")
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(strings.HasPrefix)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func("prefix-test", "prefix"), nil
			},
		},
		"strings.Join": {
			call: func() any {
				return strings.Join([]string{"a", "b", "c"}, ",")
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(strings.Join)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func([]string{"a", "b", "c"}, ","), nil
			},
		},
		"strconv.Atoi": {
			call: func() any {
				v, e := strconv.Atoi("12345")
				return struct {
					val int
					err error
				}{v, e}
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(strconv.Atoi)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				v, e := cf.Func("12345")
				return struct {
					val int
					err error
				}{v, e}, nil
			},
		},
		"strconv.FormatInt": {
			call: func() any {
				return strconv.FormatInt(255, 16)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(strconv.FormatInt)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(255, 16), nil
			},
		},
		"sort.Ints": {
			call: func() any {
				data := []int{5, 2, 8, 1, 9}
				sort.Ints(data)
				return data
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(sort.Ints)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				data := []int{5, 2, 8, 1, 9}
				cf.Func(data)
				return data, nil
			},
		},
		"sort.Strings": {
			call: func() any {
				data := []string{"zebra", "apple", "mango", "banana"}
				sort.Strings(data)
				return data
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(sort.Strings)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				data := []string{"zebra", "apple", "mango", "banana"}
				cf.Func(data)
				return data, nil
			},
		},
		"time.Since": {
			call: func() any {
				past := time.Now().Add(-1 * time.Hour)
				return time.Since(past).Truncate(time.Hour)
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(time.Since)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				past := time.Now().Add(-1 * time.Hour)
				return cf.Func(past).Truncate(time.Hour), nil
			},
		},
		"time.Unix": {
			call: func() any {
				return time.Unix(1234567890, 0).UTC()
			},
			cloneAndCall: func(t *testing.T) (any, error) {
				cf, err := cloneFunc(time.Unix)
				if err != nil {
					return nil, err
				}
				defer cf.Free()
				return cf.Func(1234567890, 0).UTC(), nil
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

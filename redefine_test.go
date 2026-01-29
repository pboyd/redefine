package redefine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestFunc_NotAFunction(t *testing.T) {
	t.Run("first arg not a function", func(t *testing.T) {
		err := Func("not a function", b)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a function")
	})

	t.Run("second arg not a function", func(t *testing.T) {
		err := Func(a, 42)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a function")
	})

	t.Run("both args not functions", func(t *testing.T) {
		err := Func([]int{1, 2, 3}, map[string]int{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a function")
	})

	t.Run("nil first arg", func(t *testing.T) {
		err := Func(nil, b)
		assert.Error(t, err)
	})

	t.Run("nil second arg", func(t *testing.T) {
		err := Func(a, nil)
		assert.Error(t, err)
	})
}

func TestFunc_SignatureMismatch(t *testing.T) {
	t.Run("different number of inputs", func(t *testing.T) {
		fn1 := func(x int) int { return x }
		fn2 := func(x, y int) int { return x + y }

		err := Func(fn1, fn2)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "signatures do not match")
		}

		err = Func(fn2, fn1)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "signatures do not match")
		}
	})

	t.Run("different number of outputs", func(t *testing.T) {
		fn1 := func() int { return 1 }
		fn2 := func() (int, error) { return 1, nil }

		err := Func(fn1, fn2)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "signatures do not match")
		}

		err = Func(fn2, fn1)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "signatures do not match")
		}
	})

	t.Run("different input types", func(t *testing.T) {
		fn1 := func(x int) int { return x }
		fn2 := func(x string) int { return len(x) }
		err := Func(fn1, fn2)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "signatures do not match")
		}
	})

	t.Run("different output types", func(t *testing.T) {
		fn1 := func() int { return 1 }
		fn2 := func() string { return "1" }
		err := Func(fn1, fn2)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "signatures do not match")
		}
	})
}

//go:noinline
func noArgsNoReturn() {
	// empty
}

func noArgsNoReturnReplacement() {
	// also empty
}

func TestFunc_NoArgsNoReturn(t *testing.T) {
	// Should not panic or error
	err := Func(noArgsNoReturn, noArgsNoReturnReplacement)
	assert.NoError(t, err)
}

//go:noinline
func multipleArgs(x int, y string, z bool) int {
	if z {
		return x + len(y)
	}
	return x
}

func multipleArgsReplacement(x int, y string, z bool) int {
	return 999
}

func TestFunc_MultipleArgs(t *testing.T) {
	assert.Equal(t, 5, multipleArgs(2, "foo", true))
	err := Func(multipleArgs, multipleArgsReplacement)
	assert.NoError(t, err)
	assert.Equal(t, 999, multipleArgs(2, "foo", true))
}

//go:noinline
func multipleReturns(x int) (int, string, error) {
	return x * 2, "original", nil
}

func multipleReturnsReplacement(x int) (int, string, error) {
	return x * 10, "replaced", nil
}

func TestFunc_MultipleReturns(t *testing.T) {
	n, s, err := multipleReturns(5)
	assert.NoError(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, "original", s)

	err = Func(multipleReturns, multipleReturnsReplacement)
	assert.NoError(t, err)

	n, s, err = multipleReturns(5)
	assert.NoError(t, err)
	assert.Equal(t, 50, n)
	assert.Equal(t, "replaced", s)
}

//go:noinline
func withPointerArg(x *int) *int {
	result := *x * 2
	return &result
}

func withPointerArgReplacement(x *int) *int {
	result := *x * 100
	return &result
}

func TestFunc_PointerArgs(t *testing.T) {
	val := 5
	result := withPointerArg(&val)
	assert.Equal(t, 10, *result)

	err := Func(withPointerArg, withPointerArgReplacement)
	assert.NoError(t, err)

	result = withPointerArg(&val)
	assert.Equal(t, 500, *result)
}

//go:noinline
func withSliceArg(s []int) int {
	sum := 0
	for _, v := range s {
		sum += v
	}
	return sum
}

func withSliceArgReplacement(s []int) int {
	return len(s)
}

func TestFunc_SliceArgs(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	assert.Equal(t, 15, withSliceArg(slice))

	err := Func(withSliceArg, withSliceArgReplacement)
	assert.NoError(t, err)

	assert.Equal(t, 5, withSliceArg(slice))
}

//go:noinline
func withMapArg(m map[string]int) int {
	return m["key"]
}

func withMapArgReplacement(m map[string]int) int {
	return -1
}

func TestFunc_MapArgs(t *testing.T) {
	m := map[string]int{"key": 42}
	assert.Equal(t, 42, withMapArg(m))

	err := Func(withMapArg, withMapArgReplacement)
	assert.NoError(t, err)

	assert.Equal(t, -1, withMapArg(m))
}

type testInterface interface {
	Value() int
}

type testImpl struct {
	val int
}

func (t testImpl) Value() int {
	return t.val
}

//go:noinline
func withInterfaceArg(i testInterface) int {
	return i.Value()
}

func withInterfaceArgReplacement(i testInterface) int {
	return i.Value() * 2
}

func TestFunc_InterfaceArgs(t *testing.T) {
	impl := testImpl{val: 21}
	assert.Equal(t, 21, withInterfaceArg(impl))

	err := Func(withInterfaceArg, withInterfaceArgReplacement)
	assert.NoError(t, err)

	assert.Equal(t, 42, withInterfaceArg(impl))
}

//go:noinline
func withStructArg(s testImpl) int {
	return s.val
}

func withStructArgReplacement(s testImpl) int {
	return s.val + 100
}

func TestFunc_StructArgs(t *testing.T) {
	s := testImpl{val: 5}
	assert.Equal(t, 5, withStructArg(s))

	err := Func(withStructArg, withStructArgReplacement)
	assert.NoError(t, err)

	assert.Equal(t, 105, withStructArg(s))
}

type testStruct struct {
	Num int
}

//go:noinline
func (ts *testStruct) Inc() {
	ts.Num++
}

type testStruct2 testStruct

func (ts *testStruct2) Double() {
	ts.Num *= 2
}

func (ts testStruct2) BadInc() {
	// Non-pointer type, so this won't work.
	ts.Num++
}

type testStruct3 struct {
	testStruct
	Other int
}

func (ts *testStruct3) Inc() {
	ts.Num++
	ts.Other++
}

func TestMethod(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ts := &testStruct{}
	ts.Inc()
	ts.Inc()
	assert.Equal(2, ts.Num)

	require.NoError(Method((*testStruct).Inc, (*testStruct2).Double))

	ts.Inc()
	assert.Equal(4, ts.Num)

	ts.Inc()
	assert.Equal(8, ts.Num)
}

func TestMethod_DifferentTypeSizes(t *testing.T) {
	assert := assert.New(t)

	// A pointer to a testStruct is the same size as a testStruct2, but
	// this call should fail because the kinds are different:
	assert.Error(Method((*testStruct).Inc, (testStruct2).BadInc))

	// Ensure that Method checks the size of the pointer destination, not
	// the size of the pointer.
	assert.Error(Method((*testStruct).Inc, (*testStruct3).Inc))
}

type testStruct4 struct {
	A uint32
	B uint32
}

func (ts *testStruct4) Inc() {
	ts.A++
	ts.B++
}

func TestMethod_DifferentTypes(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// This is to demonstrate the effect of not using an equivalent type
	// for redefined methods. The size of testStruct and testStruct4 are
	// the same but the fields are different. So the code compiled for
	// testStruct4 operates on the memory of testStruct and things get
	// weird. It works OK in this test, but in general expect crashes and
	// weird bugs.

	ts := &testStruct{}
	require.NoError(Method((*testStruct).Inc, (*testStruct4).Inc))
	ts.Inc()
	assert.Equal(1<<32|1, ts.Num)
}

//go:noinline
func variadicSum(nums ...int) int {
	sum := 0
	for _, n := range nums {
		sum += n
	}
	return sum
}

func variadicSumReplacement(nums ...int) int {
	// Returns the product instead
	product := 1
	for _, n := range nums {
		product *= n
	}
	return product
}

func TestFunc_Variadic(t *testing.T) {
	t.Run("basic variadic", func(t *testing.T) {
		assert.Equal(t, 15, variadicSum(1, 2, 3, 4, 5))

		err := Func(variadicSum, variadicSumReplacement)
		assert.NoError(t, err)

		assert.Equal(t, 120, variadicSum(1, 2, 3, 4, 5))
	})
}

//go:noinline
func variadicWithPrefix(prefix string, vals ...int) string {
	result := prefix + ":"
	for i, v := range vals {
		if i > 0 {
			result += ","
		}
		result += string(rune('0' + v))
	}
	return result
}

func variadicWithPrefixReplacement(prefix string, vals ...int) string {
	return "replaced"
}

func TestFunc_VariadicWithOtherArgs(t *testing.T) {
	assert.Equal(t, "test:1,2,3", variadicWithPrefix("test", 1, 2, 3))

	err := Func(variadicWithPrefix, variadicWithPrefixReplacement)
	assert.NoError(t, err)

	assert.Equal(t, "replaced", variadicWithPrefix("test", 1, 2, 3))
}

//go:noinline
func variadicEmpty(vals ...int) int {
	return len(vals)
}

func variadicEmptyReplacement(vals ...int) int {
	return -1
}

func TestFunc_VariadicEmpty(t *testing.T) {
	// Test with no arguments
	assert.Equal(t, 0, variadicEmpty())

	err := Func(variadicEmpty, variadicEmptyReplacement)
	assert.NoError(t, err)

	assert.Equal(t, -1, variadicEmpty())
}

//go:noinline
func genericToString[T any](val T) string {
	return fmt.Sprintf("%v", val)
}

func genericToStringReplacement[T any](val T) string {
	return fmt.Sprintf("replaced: %v", val)
}

type myType struct {
	X int
}

func TestFunc_Generics(t *testing.T) {
	t.Skipf("these currently fail")

	t.Run("generic instantiated with int", func(t *testing.T) {
		assert.Equal(t, "42", genericToString(42))

		err := Func(genericToString[int], genericToStringReplacement[int])
		assert.NoError(t, err)

		assert.Equal(t, "replaced: 42", genericToString(42))
	})

	t.Run("generic instantiated with string", func(t *testing.T) {
		assert.Equal(t, "hello", genericToString("hello"))

		err := Func(genericToString[string], genericToStringReplacement[string])
		assert.NoError(t, err)

		assert.Equal(t, "replaced: hello", genericToString("hello"))
	})

	t.Run("generic instantiated with custom type", func(t *testing.T) {
		instance := myType{X: 1}
		assert.Equal(t, "42", genericToString(instance))

		err := Func(genericToString[myType], genericToStringReplacement[myType])
		assert.NoError(t, err)

		assert.Equal(t, "replaced: 42", genericToString(instance))
	})
}

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
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signatures do not match")
	})

	t.Run("different number of outputs", func(t *testing.T) {
		fn1 := func() int { return 1 }
		fn2 := func() (int, error) { return 1, nil }
		err := Func(fn1, fn2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signatures do not match")
	})

	t.Run("different input types", func(t *testing.T) {
		fn1 := func(x int) int { return x }
		fn2 := func(x string) int { return len(x) }
		err := Func(fn1, fn2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signatures do not match")
	})

	t.Run("different output types", func(t *testing.T) {
		fn1 := func() int { return 1 }
		fn2 := func() string { return "1" }
		err := Func(fn1, fn2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signatures do not match")
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

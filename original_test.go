//go:build amd64

package redefine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuncWithOriginal(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("a", a())
	assert.NoError(Func(a, b))
	assert.Equal("b", a())

	assert.Equal("a", Original(a)())

	assert.NoError(Restore(a))
	assert.Equal("a", a())
}

func TestMethodWithOriginal(t *testing.T) {
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

	Original((*testStruct).Inc)(ts)
	assert.Equal(9, ts.Num)

	ts.Inc()
	assert.Equal(18, ts.Num)

	assert.NoError(Restore((*testStruct).Inc))
	ts.Inc()
	assert.Equal(19, ts.Num)
}

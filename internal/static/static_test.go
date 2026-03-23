package static

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuncSlice(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	info := GetInfo()

	buf, err := info.FuncSlice(testFunc)
	require.NoError(err)

	assert.Greater(len(buf), 4)
}

func testFunc() int {
	return 5
}

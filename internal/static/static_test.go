package static

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuncSlice(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	info := GetInfo()
	buf, addr, err := info.FuncSlice(testFunc)
	require.NoError(err)

	assert.Greater(len(buf), 4)

	assert.Equal(uintptr(unsafe.Pointer(unsafe.SliceData(buf)))-info.writeOffset, addr)
}

func testFunc() int {
	return 5
}

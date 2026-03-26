//go:build darwin && arm64

package static

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWritableText(t *testing.T) {
	assert := assert.New(t)

	info := GetInfo()

	offset, err := info.getWriteOffset()
	require.NoError(t, err)

	assert.NotEqual(uintptr(0), offset)
	assert.Equal(offset, info.writeOffset)

	ptr := reflect.ValueOf(editable).Pointer()
	execSlice := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), 4)
	editSlice := unsafe.Slice((*byte)(unsafe.Pointer(ptr+offset)), 4)

	// The content should be the same
	assert.Equal(editSlice, execSlice)

	editSlice[0] = 0
	editSlice[1] = 1
	editSlice[2] = 2
	editSlice[3] = 3

	// The content should still be the same after changing the editable copy
	assert.Equal(editSlice, execSlice)
}

func editable() int {
	return 3
}

//go:build darwin && arm64

package static

import (
	"reflect"
	"strconv"
	"testing"
	"unsafe"

	"github.com/pboyd/redefine/internal/mach"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuplicate(t *testing.T) {
	staticVar = 0

	info := GetInfo()
	t.Log("Original (before duplicate)\n" + info.String())

	dupInfo, err := info.duplicate()
	require.NoError(t, err)

	t.Log("Duplicate\n" + dupInfo.String())

	assert.NotEqual(t, uintptr(0), info.offset)

	t.Run("static data function", func(t *testing.T) {
		assert := assert.New(t)

		// Sanity check that staticDataFunc works.
		assert.Equal(1, staticDataFunc())
		assert.Equal(2, staticDataFunc())

		// Now get the copy of staticDataFunc in the duplicated text
		// segment. It should update the same static data.
		dup := offsetFunc(staticDataFunc, info.offset)

		assert.Equal(3, dup())
		assert.Equal(4, dup())

		// Finally, make sure that the adds done by dup are reflected
		// in the original.
		assert.Equal(5, staticDataFunc())
	})

	t.Run("stack allocation", func(t *testing.T) {
		assert := assert.New(t)
		dup := offsetFunc(stackFunc, info.offset)
		assert.Equal(1, dup(0))
	})

	t.Run("findfunc", func(t *testing.T) {
		assert := assert.New(t)

		dupFindfunc := offsetFunc(findfunc, info.offset)

		fi := dupFindfunc(reflect.ValueOf(staticDataFunc).Pointer() + info.offset)
		assert.NotNil(fi._func)
		assert.NotNil(fi.datap)
	})

	t.Run("split stack", func(t *testing.T) {
		assert := assert.New(t)

		g := getg()
		origLo := g.stack.lo

		// Sanity check: stackSplitter really splits the stack
		stackSplitter(0, 0)
		assert.NotEqual(origLo, g.stack.lo)

		// Another sanity check: we get the same g instance no matter
		// where the function runs.
		g2 := offsetFunc(getg, info.offset)()
		assert.Same(g, g2)

		newLo := g.stack.lo

		offsetFunc(stackSplitter, info.offset)(0, 0)
		assert.NotEqual(newLo, g.stack.lo)
	})

	t.Run("FuncSlice", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		text, etext := dupInfo.Text()

		buf, err := info.FuncSlice(staticDataFunc)
		require.NoError(err)

		assert.Greater(len(buf), 4)

		bufAddr := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))
		assert.True(bufAddr >= text && bufAddr < etext, "pointer to original function in the original instance should still get the duplicate data")

		dupBuf, err := dupInfo.FuncSlice(staticDataFunc)
		assert.NoError(err)

		dupBufAddr := uintptr(unsafe.Pointer(unsafe.SliceData(dupBuf)))
		assert.Equal(bufAddr, dupBufAddr, "pointer to original function in the duplicate instance should get the duplicate data")

		dupFunc := offsetFunc(staticDataFunc, info.offset)

		dupFuncBuf, err := dupInfo.FuncSlice(dupFunc)
		assert.NoError(err)

		dupFuncBufAddr := uintptr(unsafe.Pointer(unsafe.SliceData(dupFuncBuf)))
		assert.Equal(bufAddr, dupFuncBufAddr, "pointer to duplicate function in the duplicate instance should get the duplicate data")
	})

	t.Run("Duplicated Text Edits", func(t *testing.T) {
		buf, err := info.FuncSlice(uncalledFunc)
		require.NoError(t, err)

		// Make sure writes to the duplicated text data don't panic if
		// JITWriteUnprotect is called first.
		mach.JITWriteUnprotect()
		buf[0] = 0
		mach.JITWriteProtect()
	})
}

var refs []any

// offsetFunc takes the address of fn and adds offset to it, then derefs that
// address as a function of the same type.
func offsetFunc[T any](fn T, offset uintptr) T {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		panic("not a function")
	}

	ptr := fnv.Pointer() + offset
	ref := &ptr
	refs = append(refs, ref)

	return *(*T)(unsafe.Pointer(uintptr(unsafe.Pointer(&ref))))
}

var staticVar int

//go:noinline
func staticDataFunc() int {
	staticVar++
	return staticVar
}

// stackSplitter calls itself recursively until the stack grows.
//
//go:noinline
func stackSplitter(n int, lo uintptr) int {
	g := getg()
	if lo == 0 {
		lo = g.stack.lo
	} else if g.stack.lo != lo {
		return n
	}

	return stackSplitter(n+1, lo)
}

func stackFunc(n int) int {
	str := strconv.Itoa(n)
	return len(str)
}

func uncalledFunc() int {
	return 3
}

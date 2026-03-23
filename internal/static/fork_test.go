//go:build darwin && arm64

package static

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testForkStatic any

// forkTestFuncVal is a package-level function variable whose funcval lives in
// rodata.  patchRodataCodePtrs must patch its code pointer to the duplicate
// text after Fork().
//
//go:noinline
func forkTestHelper() int { return 17 }

var forkTestFuncVal func() int = forkTestHelper

func TestFork(t *testing.T) {
	testForkStatic = int(5)

	assert.False(t, runningInDuplicate())

	info := GetInfo()

	err := Fork()
	require.NoError(t, err)
	assert.NotNil(t, info.dupInfo)

	assert.True(t, runningInDuplicate())

	t.Run("goroutines", func(t *testing.T) {
		assert.True(t, runningInDuplicate())

		ch := make(chan bool, 1)
		go func() {
			defer close(ch)
			ch <- runningInDuplicate()
		}()
		assert.True(t, <-ch)
	})

	t.Run("type assertions", func(t *testing.T) {
		v, ok := testForkStatic.(int)
		assert.True(t, ok)
		assert.Equal(t, 5, v)
	})

	t.Run("funcval dispatch after fork", func(t *testing.T) {
		// A Go func value is a pointer to a funcval whose first word is
		// the code entry address.  For a package-level function variable
		// the funcval lives in rodata, so patchRodataCodePtrs should
		// have updated its code pointer to the dupInfo text.
		dupText, dupEtext := info.dupInfo.Text()

		// Dereference the func value to get the funcval pointer, then
		// read the first word (the code pointer).
		fvPtr := *(*uintptr)(unsafe.Pointer(&forkTestFuncVal))
		codePtr := *(*uintptr)(unsafe.Pointer(fvPtr))

		assert.True(t, codePtr >= dupText && codePtr < dupEtext,
			"funcval code pointer 0x%x should be in duplicate text [0x%x, 0x%x) after Fork()",
			codePtr, dupText, dupEtext)

		// The funcval must still dispatch correctly.
		assert.Equal(t, 17, forkTestFuncVal())
	})
}

func TestFrame(t *testing.T) {
	assert := assert.New(t)

	f := getFrame()
	assert.Greater(f.lr, lastmoduledatap.minpc)
	assert.Less(f.lr, lastmoduledatap.maxpc)

	// Check the name of the function that called this test. It may change
	// in the future which would break this test.
	fn := f.Func()
	assert.Equal("testing.tRunner", fn.Name())

	// There should at least be one additional caller.
	f = f.next
	assert.NotNil(f)

	for ; f != nil; f = f.next {
		t.Logf("name=%s", f.Func().Name())
		assert.Greater(f.lr, lastmoduledatap.minpc)
		assert.Less(f.lr, lastmoduledatap.maxpc)
	}
}

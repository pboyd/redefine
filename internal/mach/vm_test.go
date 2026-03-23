//go:build darwin && arm64

package mach

import (
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

var pageSize = uintptr(syscall.Getpagesize())

func TestRemap(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// The scenario is two source pages and two destination pages. The
	// second destination page is remapped from the source. This tests the
	// way we intend to use this function.
	src, err := unix.MmapPtr(-1, 0, nil, 2*pageSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	require.NoError(err)
	defer unix.MunmapPtr(src, 2*pageSize)

	srcBuf := unsafe.Slice((*byte)(src), 2*pageSize)
	for i := range srcBuf {
		srcBuf[i] = byte(i)
	}

	dest, err := unix.MmapPtr(-1, 0, nil, pageSize, unix.PROT_NONE, unix.MAP_ANON|unix.MAP_PRIVATE|unix.MAP_JIT)
	require.NoError(err)
	defer unix.MunmapPtr(dest, 2*pageSize)

	destPage1 := unsafe.Slice((*byte)(dest), pageSize)
	err = unix.Mprotect(destPage1, unix.PROT_READ|unix.PROT_WRITE)
	require.NoError(err)

	for i := range destPage1 {
		destPage1[i] = ^byte(i)
	}

	remapAddr := uintptr(dest) + pageSize
	info, err := VmRemap(remapAddr, uintptr(src)+pageSize, pageSize)
	require.NoError(err)

	defer info.Unmap()

	assert.Equal(remapAddr, uintptr(info.Addr))
	assert.Equal(pageSize, info.Size)
	assert.Equal(info.Prot, VmProtRead|VmProtWrite)
	assert.Equal(info.MaxProt, VmProtRead|VmProtWrite|VmProtExecute)

	destPage2 := info.Slice()
	srcPage2 := srcBuf[pageSize:]

	t.Logf("src=0x%x-0x%x dest=0x%x-0x%x", uintptr(src)+pageSize, uintptr(src)+2*pageSize, remapAddr, remapAddr+pageSize)

	assert.Equal(srcPage2, destPage2)

	destPage2[0] = 0x12
	assert.Equal(destPage2[0], srcPage2[0], "writes to dest page are reflected in the source page")

	srcPage2[100] = 0xff
	assert.Equal(srcPage2[100], destPage2[100], "writes to source page are reflected in the dest page")
}

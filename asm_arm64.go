package redefine

import (
	"encoding/binary"
	"errors"
	"unsafe"
)

func insertJump(buf []byte, dest uintptr) error {
	if len(buf) < 4 {
		return errors.New("buffer too small")
	}

	addr := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))
	offset := int32(dest - addr)

	// Encode the instruction:
	// -----------------------------------
	// | 000101 | ... 26 bit address ... |
	// -----------------------------------
	inst := (5 << 26) | (uint32(offset>>2) & (1<<26 - 1))
	binary.LittleEndian.PutUint32(buf, inst)

	// Pad the rest of the buffer with nulls
	for i := 4; i < len(buf); i++ {
		buf[i] = 0
	}

	return nil
}

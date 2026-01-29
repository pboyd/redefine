package redefine

import (
	"encoding/binary"
	"unsafe"
)

func insertJump(buf []byte, dest uintptr) error {
	// Address to jump from. Add 5 because the instruction takes 5 bytes (1 byte opcode, 4 byte address).
	src := uintptr(unsafe.Pointer(unsafe.SliceData(buf))) + 5

	buf[0] = 0xe9 // JMP rel32
	diff32 := int32(dest - src)
	binary.LittleEndian.PutUint32(buf[1:], uint32(diff32))

	return nil
}

package redefine

import (
	"encoding/binary"
	"errors"
	"unsafe"
)

func insertJump(buf []byte, dest uintptr) error {
	const instructionSize = 5 // 1 byte opcode + 4 byte address

	// Make sure the buffer has enough space. As far as I can tell, there
	// should always be at least 32 bytes to work with, but it doesn't hurt
	// to check.
	if len(buf) < instructionSize {
		return errors.New("buffer to small for jump instruction")
	}

	// Address to jump from
	src := uintptr(unsafe.Pointer(unsafe.SliceData(buf))) + instructionSize

	buf[0] = 0xe9 // JMP rel32
	diff32 := int32(dest - src)
	binary.LittleEndian.PutUint32(buf[1:], uint32(diff32))

	// Pad the rest of the buffer INT3 opcodes to match what the compiler does
	for i := instructionSize; i < len(buf); i++ {
		buf[i] = 0xcc // INT3
	}

	return nil
}

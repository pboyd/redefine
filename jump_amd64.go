package redefine

import (
	"encoding/binary"
	"reflect"
	"unsafe"
)

func insertJump(buf []byte, dest uintptr) error {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&buf))

	// Address to jump from. Add 5 because the instruction takes 5 bytes (1 byte opcode, 4 byte address).
	src := sliceHeader.Data + 5

	buf[0] = 0xe9 // JMP rel32
	diff32 := int32(dest - src)
	binary.LittleEndian.PutUint32(buf[1:], uint32(diff32))

	return nil
}

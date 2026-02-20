package redefine

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"unsafe"

	"golang.org/x/arch/x86/x86asm"
)

const (
	opcodeCALLabs = 0xff // CALL abs32
	opcodeCALLrel = 0xe8 // CALL rel32
	opcodeINT3    = 0xcc
	opcodeJMP     = 0xe9 // JMP rel32
)

func insertJump(buf []byte, dest uintptr) error {
	const instructionSize = 5 // 1 byte opcode + 4 byte address

	// Make sure the buffer has enough space. As far as I can tell, there
	// should always be at least 32 bytes to work with, but it doesn't hurt
	// to check.
	if len(buf) < instructionSize {
		return errors.New("buffer too small for jump instruction")
	}

	// Address to jump from
	src := uintptr(unsafe.Pointer(unsafe.SliceData(buf))) + instructionSize

	buf[0] = opcodeJMP
	diff32 := int32(dest - src)
	binary.LittleEndian.PutUint32(buf[1:], uint32(diff32))

	// Pad the rest of the buffer with INT3 opcodes to match what the compiler does
	for i := instructionSize; i < len(buf); i++ {
		buf[i] = opcodeINT3
	}

	return nil
}

// relocateFunc copies machine instructions from src into dest translating
// relative instructions as it goes. dest must be larger than src.
//
// The data underlying the slices is assumed to be the same address the code
// would execute from.
//
// The dest slice is returned after being resized.
func relocateFunc(src, dest []byte) ([]byte, error) {
	dest = dest[:len(src)]

	for i := 0; i < len(src); {
		instruction, err := x86asm.Decode(src[i:], 64)
		if err != nil {
			return nil, fmt.Errorf("decode error at offset %d: %w", i, err)
		}

		copy(dest[i:], src[i:i+instruction.Len])

		if instruction.PCRel > 0 {
			err = fixPCRelAddress(instruction, src[i:i+instruction.Len], dest[i:i+instruction.Len])
			if err != nil {
				return nil, err
			}
		}

		i += instruction.Len
	}

	return dest, nil
}

func fixPCRelAddress(inst x86asm.Inst, src, dest []byte) error {
	srcPC := uintptr(unsafe.Pointer(unsafe.SliceData(src))) + uintptr(len(src))
	destPC := uintptr(unsafe.Pointer(unsafe.SliceData(dest))) + uintptr(len(dest))

	switch inst.PCRel {
	case 4:
		disp := int32(binary.LittleEndian.Uint32(src[inst.PCRelOff:]))
		newDisp := (int64(srcPC) + int64(disp)) - int64(destPC)
		if newDisp < math.MinInt32 || newDisp > math.MaxInt32 {
			return fmt.Errorf("error at address srcPC=0x%x destPC=0x%x: unable to translate relative address (%d overflows int32)", srcPC, destPC, newDisp)
		}

		binary.LittleEndian.PutUint32(dest[inst.PCRelOff:], uint32(newDisp))
	case 1:
		// Ignore 1-byte relative addresses because their most likely jumps inside the function.
	default:
		return fmt.Errorf("unsupported relative address size: %d", inst.PCRel)
	}

	return nil
}

func disassemble(code []byte) (string, error) {
	var buf bytes.Buffer

	baseAddr := uintptr(unsafe.Pointer(unsafe.SliceData(code)))

	for i := 0; i < len(code); {
		instruction, err := x86asm.Decode(code[i:], 64)
		if err != nil {
			return "", fmt.Errorf("decode error at offset %d: %w", i, err)
		}
		fmt.Fprintf(&buf, "0x%08x\t%-20s\t%s\n", baseAddr+uintptr(i), hex.EncodeToString(code[i:i+instruction.Len]), instruction.String())

		i += instruction.Len
	}

	return buf.String(), nil
}

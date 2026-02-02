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
	opcodeJMP     = 0xe9 // JMP rel32
	opcodeINT3    = 0xcc
	opcodeCALLrel = 0xe8 // CALL rel32
	opcodeCALLabs = 0xff // CALL abs32
	opcodeMOVimm  = 0xc7 // MOV imm, r/m

	regModeDirect = 3
	registerBP    = 5
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

	// Pad the rest of the buffer INT3 opcodes to match what the compiler does
	for i := instructionSize; i < len(buf); i++ {
		buf[i] = opcodeINT3
	}

	return nil
}

// relocateFunc copies machine instructions from src into a new buffer,
// translating relative CALL instructions as it goes.
//
// The data for the underlying array in src is assumed to be at the same
// address the code would execute from.
func relocateFunc(src []byte) ([]byte, error) {
	baseAddress := uintptr(unsafe.Pointer(unsafe.SliceData(src)))

	padStart := len(src) - 1
	for ; src[padStart] == opcodeINT3; padStart-- {
	}
	src = src[:padStart+1]

	dest := make([]byte, len(src))

	for i := 0; i < len(src); {
		instruction, err := x86asm.Decode(src[i:], 64)
		if err != nil {
			return nil, fmt.Errorf("decode error at offset %d: %w", i, err)
		}

		if instruction.Opcode>>24 == opcodeCALLrel {
			rel, ok := instruction.Args[0].(x86asm.Rel)
			if !ok {
				return nil, fmt.Errorf("decode error at offset %d: unknown argument", i)
			}

			absCallDest := uintptr(int32(baseAddress+uintptr(i)+uintptr(instruction.Len)) + int32(rel))
			jumpBack := int32(i + instruction.Len - len(dest))
			ccBuf, err := callCode(absCallDest, jumpBack)
			if err != nil {
				return nil, fmt.Errorf("unable to generate call code: %w", err)
			}
			jumpTo := int32(len(dest) - (i + instruction.Len))

			dest = append(dest, ccBuf...)

			dest[i] = opcodeJMP
			binary.LittleEndian.PutUint32(dest[i+1:], uint32(jumpTo))
		} else {
			copy(dest[i:], src[i:i+instruction.Len])
		}

		i += instruction.Len
	}

	padding := make([]byte, ((len(dest)+0xf)&^0xf)-len(dest))
	for i := range padding {
		padding[i] = opcodeINT3
	}
	dest = append(dest, padding...)

	return dest, nil
}

// callCode returns the x86-64 machine code equivalent of:
//
//	MOVQ <callDest>, BP
//	CALL BP
//	JMP <jumpBack+offset>
//
// jumpBack should be relative to the beginning of the block and will be
// adjusted for it's final address.
func callCode(callDest uintptr, jumpBack int32) ([]byte, error) {
	if callDest > math.MaxUint32 {
		// TODO: Should this support 64-bit addresses?
		return nil, errors.New("64-bit call is not implemented")
	}

	buf := make([]byte, 14)
	i := 0

	// MOVQ <callDest> BP
	buf[i] = byte(x86asm.PrefixREX) | byte(x86asm.PrefixREXW)
	i++
	buf[i] = opcodeMOVimm
	i++
	buf[i] = regModeDirect<<6 | registerBP
	i++

	binary.LittleEndian.PutUint32(buf[i:], uint32(callDest))
	i += 4

	// CALL BP
	buf[i] = opcodeCALLabs
	i++
	buf[i] = regModeDirect<<6 | 2<<3 | registerBP
	i++

	// JMP <jumpBack>
	buf[i] = opcodeJMP
	i++
	binary.LittleEndian.PutUint32(buf[i:], uint32(jumpBack-int32(i)-4))
	i += 4

	return buf, nil
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

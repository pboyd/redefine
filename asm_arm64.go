package redefine

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/arch/arm64/arm64asm"
)

const (
	// -----------------------------------
	// | 000101 | ... 26 bit address ... |
	// -----------------------------------
	_B = uint32(5 << 26)

	// -----------------------------------
	// | 100101 | ... 26 bit address ... |
	// -----------------------------------
	_BL = uint32(1<<31 | _B)

	// ADR/ADRP is encoded as:
	// --------------------------------------------------
	// | P | lo 2 bits | 10000 | hi 19 bits | 5-bit reg |
	// --------------------------------------------------
	// Mask for the address:
	adrAddressMask = uint32(3<<29 | 0x7ffff<<5)
)

// The maximum acceptable distance from the text and data segments.
const maxCloneDistance = 128 * 1024 * 1024

func insertJump(buf []byte, dest uintptr) error {
	if len(buf) < 4 {
		return errors.New("buffer too small")
	}

	addr := int64(uintptr(unsafe.Pointer(unsafe.SliceData(buf))))
	offset := int64(dest) - addr

	if offset < -(1<<27) || offset >= (1<<27) {
		return fmt.Errorf("B target out of range: %d bytes exceeds 128MiB", offset)
	}

	inst := _B | (uint32(offset>>2) & (1<<26 - 1))
	binary.LittleEndian.PutUint32(buf, inst)

	// Pad the rest of the buffer with nulls
	for i := 4; i < len(buf); i++ {
		buf[i] = 0
	}

	return nil
}

// relocateFunc copies machine instructions from src into dest translating
// relative instructions as it goes. dest must be at least as large as src.
//
// The data underlying the slices is assumed to be the same address the code
// would execute from.
func relocateFunc(src, dest []byte) ([]byte, error) {
	dest = dest[:len(src)]
	copy(dest, src)

	srcPC := uintptr(unsafe.Pointer(unsafe.SliceData(src)))

	for i := 0; i < len(src); i += 4 {
		raw := dest[i : i+4]

		instruction, err := arm64asm.Decode(raw)
		if err != nil {
			// Stop if the bad instruction was padding
			if bytes.Equal(raw, []byte{0, 0, 0, 0}) {
				break
			}
			return nil, fmt.Errorf("decode error at offset %d %v: %w", i, raw, err)
		}

		for _, arg := range instruction.Args {
			if _, ok := arg.(arm64asm.PCRel); ok {
				err = fixPCRelAddress(instruction, srcPC, raw)
				if err != nil {
					return nil, err
				}
			}
		}
		srcPC += 4
	}

	return dest, nil
}

func fixPCRelAddress(inst arm64asm.Inst, srcPC uintptr, dest []byte) error {
	destPC := uintptr(unsafe.Pointer(unsafe.SliceData(dest)))

	switch inst.Op {
	case arm64asm.ADRP:
		// Get the offset (arm64asm converts it to bytes)
		oldOffset := int64(inst.Args[1].(arm64asm.PCRel))

		// Page-align both addresses before computing the offset
		newOffsetPages := (int64(srcPC&^uintptr(0xfff)) + oldOffset - int64(destPC&^uintptr(0xfff))) >> 12

		if newOffsetPages < -(1<<20) || newOffsetPages >= (1<<20) {
			return fmt.Errorf("ADRP target out of range: %d pages exceeds 4GiB", newOffsetPages)
		}

		p := uint32(newOffsetPages)
		encoded := binary.LittleEndian.Uint32(dest) &^ adrAddressMask
		encoded |= (p & 3) << 29 // Lowest 2 bits to bits 30 and 29
		encoded |= (p >> 2) << 5 // Highest 19 bits to bits 23 to 5
		binary.LittleEndian.PutUint32(dest, encoded)

	case arm64asm.BL:
		oldOffset := int64(inst.Args[0].(arm64asm.PCRel))
		offset := int64(srcPC) + oldOffset - int64(destPC)

		// BL encodes a 26-bit signed instruction offset.
		if offset < -(1<<27) || offset >= (1<<27) {
			return fmt.Errorf("BL target out of range: %d bytes exceeds 128MiB", offset)
		}

		binary.LittleEndian.PutUint32(dest, _BL|(uint32(offset>>2)&(1<<26-1)))

	default:
		// Most PC-relative addresses are local. Go only seems to
		// generate ADRP and BL that are external to the function.
	}

	return nil
}

func disassemble(code []byte) (string, error) {
	var buf bytes.Buffer

	baseAddr := uintptr(unsafe.Pointer(unsafe.SliceData(code)))

	for i := 0; i < len(code)&^3; i += 4 {
		var asm string
		instruction, err := arm64asm.Decode(code[i:])
		if err == nil {
			asm = instruction.String()
		} else {
			asm = "?"
		}
		fmt.Fprintf(&buf, "0x%08x\t%-20s\t%s\n", baseAddr+uintptr(i), hex.EncodeToString(code[i:i+4]), asm)
	}

	return buf.String(), nil
}

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

	// ----------------------------------------------
	// | 1101011000111111000000 | 5-bit reg | 00000 |
	// ----------------------------------------------
	_BLR = uint32(0xd63f0000)

	// -----------------------------------------------------------
	// | 1-bit sf | 10100101 | 2-bit hw | 16-bit imm | 5-bit reg |
	// -----------------------------------------------------------
	_MOVZ = uint32(0xd2800000) // sf is 1

	// -----------------------------------------------------------
	// | 1-bit sf | 11100101 | 2-bit hw | 16-bit imm | 5-bit reg |
	// -----------------------------------------------------------
	_MOVK = uint32(0xf2800000) // sf is 1

	// ADR/ADRP is encoded as:
	// --------------------------------------------------
	// | P | lo 2 bits | 10000 | hi 19 bits | 5-bit reg |
	// --------------------------------------------------
	// Mask for the address:
	adrAddressMask = uint32(3<<29 | 0x7ffff<<5)
)

const scratchRegister = 16

// Ideally, cloned functions will be within 128 MiB of the original function.
// But it's acceptable to be within the 4 GiB range for ADRP because there's code
// to generate trampolines for BLs.
const idealCloneDistance = 128 * 1024 * 1024
const maxCloneDistance = 4 * 1024 * 1024 * 1024

func insertJump(buf []byte, dest uintptr) error {
	if len(buf) < 4 {
		return errors.New("buffer too small")
	}

	addr := int64(uintptr(unsafe.Pointer(unsafe.SliceData(buf))))
	offset := int64(dest) - addr

	if offset < -(1<<27) || offset >= (1<<27) {
		return fmt.Errorf("B target out of range: %d bytes exceeds 128MiB", offset)
	}

	encodeB(buf, int32(offset))

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
	src = trimPadding(src)
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
					if errors.Is(err, errAddressOutOfRange) && instruction.Op == arm64asm.BL {
						var trErr error
						dest, trErr = makeBLTrampoline(instruction, srcPC, dest, i)
						if trErr != nil {
							return nil, fmt.Errorf("unable to make trampoline: %w (original error: %w)", trErr, err)
						}
					} else {
						return nil, err
					}
				}
			}
		}
		srcPC += 4
	}

	return dest, nil
}

func trimPadding(buf []byte) []byte {
	newLen := len(buf)
	for i := len(buf) - 4; i >= 0; i -= 4 {
		if bytes.Equal(buf[i:i+4], []byte{0, 0, 0, 0}) {
			newLen = i
		} else {
			break
		}
	}

	return buf[:newLen]
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
			return fmt.Errorf("%w: ADRP target out of range: %d pages exceeds 4GiB", errAddressOutOfRange, newOffsetPages)
		}

		p := uint32(newOffsetPages)
		encoded := binary.LittleEndian.Uint32(dest) &^ adrAddressMask
		encoded |= (p & 3) << 29             // Lowest 2 bits to bits 30 and 29
		encoded |= ((p >> 2) & 0x7ffff) << 5 // Highest 19 bits to bits 23 to 5
		binary.LittleEndian.PutUint32(dest, encoded)

	case arm64asm.BL:
		oldOffset := int64(inst.Args[0].(arm64asm.PCRel))
		offset := int64(srcPC) + oldOffset - int64(destPC)

		// BL encodes a 26-bit signed instruction offset.
		if offset < -(1<<27) || offset >= (1<<27) {
			return fmt.Errorf("%w: BL target out of range: %d bytes exceeds 128MiB", errAddressOutOfRange, offset)
		}

		binary.LittleEndian.PutUint32(dest, _BL|(uint32(offset>>2)&(1<<26-1)))

	default:
		// Most PC-relative addresses are local. Go only seems to
		// generate ADRP and BL that are external to the function.
	}

	return nil
}

func makeBLTrampoline(inst arm64asm.Inst, srcPC uintptr, dest []byte, blOffset int) ([]byte, error) {
	if cap(dest)-len(dest) < 24 {
		return nil, errors.New("destination is too small for BL trampoline")
	}
	origLen := len(dest)
	dest = dest[:len(dest)+24]

	blrTarget := uintptr(int64(srcPC) + int64(inst.Args[0].(arm64asm.PCRel)))

	// Encode the trampoline itself. It uses 6 instructions total. 4 to
	// store a 64-bit number in x16, 1 for BLR x16, and 1 B to return the
	// caller.
	trampoline := dest[origLen:]
	encodeMov(trampoline, true, 0, uint16(blrTarget), scratchRegister)
	encodeMov(trampoline[4:], false, 16, uint16(blrTarget>>16), scratchRegister)
	encodeMov(trampoline[8:], false, 32, uint16(blrTarget>>32), scratchRegister)
	encodeMov(trampoline[12:], false, 48, uint16(blrTarget>>48), scratchRegister)
	binary.LittleEndian.PutUint32(trampoline[16:], _BLR|uint32(scratchRegister<<5))

	// Replace the original BL with a B to the beginning of the trampoline
	blAddr := uintptr(unsafe.Pointer(unsafe.SliceData(dest))) + uintptr(blOffset)
	trampolineAddr := uintptr(unsafe.Pointer(unsafe.SliceData(trampoline)))
	encodeB(dest[blOffset:], int32(int64(trampolineAddr)-int64(blAddr)))

	// The last instruction in the trampoline needs to jump back to the
	// instruction after the original BL
	encodeB(trampoline[20:], int32(int64(blAddr+4)-int64(trampolineAddr+20)))

	return dest, nil
}

func encodeB(dest []byte, offset int32) {
	inst := _B | (uint32(offset)>>2)&0x3ffffff
	binary.LittleEndian.PutUint32(dest, inst)
}

func encodeMov(dest []byte, zero bool, lsl uint8, imm uint16, register uint8) {
	var mov uint32
	if zero {
		mov = _MOVZ
	} else {
		mov = _MOVK
	}

	mov |= (uint32(lsl>>4) & 3) << 21
	mov |= uint32(imm) << 5
	mov |= uint32(register & 0x1f)

	binary.LittleEndian.PutUint32(dest, mov)
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

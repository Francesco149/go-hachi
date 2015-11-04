/*
	Copyright 2015 Franc[e]sco (lolisamurai@tfwno.gf)
	This file is part of go-hachi.
	go-hachi is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
	go-hachi is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.
	You should have received a copy of the GNU General Public License
	along with go-hachi. If not, see <http://www.gnu.org/licenses/>.
*/

package hachi

import "fmt"

// An Instruction is any decompiled CHIP-8 instruction.
type Instruction interface {
	// Returns a detailed description of what the instruction does.
	Description() string
	// Returns a pseudo-asm representation of the instruction.
	String() string
	// Returns the instruction.
	Opcode() uint16
	// Returns the size of the instruction in bytes.
	Size() int
	// Returns the ASCII representation of the raw data for this instruction.
	// Returns an empty string if the data is not printable ascii.
	ASCII() string

	init()
}

// RawData holds 1 or 2 bytes of unrecognized raw data
type RawData struct {
	Instruction
	b []byte
	s string
}

func (i RawData) init()          { i.s = fmt.Sprintf("DB % 02X", i.b) }
func (i RawData) String() string { return i.s }

// Opcode returns the data as a 16-bit integer. Normally, this function is
// used to get the opcode for an instruction, but RawData is a special case
// in which this function is not used.
func (i RawData) Opcode() (res uint16) {
	res = uint16(i.b[0])
	if len(i.b) == 2 {
		res <<= 8
		res |= uint16(i.b[1])
	}
	return
}

func (i RawData) Size() int           { return len(i.b) }
func (i RawData) Description() string { return "Unknown / Raw Data" }

func (i RawData) ASCII() (res string) {
	if isPrintableASCII(i.b) {
		res = string(i.b)
	}
	return
}

// -----------------------------------------------------------------------------

type Sys struct{ *RawData }

func (i Sys) init() {
	// opcode starts with zero so we can use the entire opcode as the addr
	i.s = fmt.Sprintf("SYS %03X", i.Address())

	switch uint16(i.b[0]&0x0F)<<8 | uint16(i.b[1]) {
	case 0x0E0:
		i.s += " (CLS)"
	case 0x0EE:
		i.s += " (RET)"
	}
}
func (i Sys) Address() uint16 { return i.Opcode() }
func (i Sys) Description() string {
	return "0NNN: Calls RCA 1802 program at address NNN."
}

//

type Jp struct{ *RawData }

func (i Jp) init()               { i.s = fmt.Sprintf("JP %03X", i.Address()) }
func (i Jp) Address() uint16     { return i.Opcode() & 0x0FFF }
func (i Jp) Description() string { return "1NNN: Jumps to address NNN." }

//

type Call struct{ *RawData }

func (i Call) init() {
	i.s = fmt.Sprintf("CALL %03X", i.Address())
}
func (i Call) Address() uint16     { return i.Opcode() & 0x0FFF }
func (i Call) Description() string { return "2NNN: Calls subroutine at NNN." }

//

type Se struct{ *RawData }

func (i Se) init() {
	i.s = fmt.Sprintf("SE V%1X,%02X", i.Register(), i.Value())
}
func (i Se) Register() uint8 { return i.b[0] & 0x0F }
func (i Se) Value() uint8    { return i.b[1] }
func (i Se) Description() string {
	return "3XNN: Skips the next instruction if VX equals NN."
}

//

type Sne struct{ *RawData }

func (i Sne) init() {
	i.s = fmt.Sprintf("SNE V%1X,%02X", i.Register(), i.Value())
}
func (i Sne) Register() uint8 { return i.b[0] & 0x0F }
func (i Sne) Value() uint8    { return i.b[1] }
func (i Sne) Description() string {
	return "4XNN: Skips the next instruction if VX doesn't equal NN."
}

//

type SeRegister struct{ *RawData }

func (i SeRegister) init() {
	i.s = fmt.Sprintf("SE V%1X,V%1X", i.Register1(), i.Register2())
}
func (i SeRegister) Register1() uint8 { return i.b[0] & 0x0F }
func (i SeRegister) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i SeRegister) Description() string {
	return "5XY0: Skips the next instruction if VX equals VY."
}

//

type Ld struct{ *RawData }

func (i Ld) init() {
	i.s = fmt.Sprintf("LD V%1X,%02X", i.Register(), i.Value())
}
func (i Ld) Register() uint8     { return i.b[0] & 0x0F }
func (i Ld) Value() uint8        { return i.b[1] }
func (i Ld) Description() string { return "6XNN: Sets VX to NN." }

//

type Add struct{ *RawData }

func (i Add) init() {
	i.s = fmt.Sprintf("ADD V%1X,%02X", i.Register(), i.Value())
}
func (i Add) Register() uint8     { return i.b[0] & 0x0F }
func (i Add) Value() uint8        { return i.b[1] }
func (i Add) Description() string { return "7XNN: Adds NN to VX." }

//

type LdRegister struct{ *RawData }

func (i LdRegister) init() {
	i.s = fmt.Sprintf("LD V%1X,V%1X", i.Register1(), i.Register2())
}
func (i LdRegister) Register1() uint8 { return i.b[0] & 0x0F }
func (i LdRegister) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i LdRegister) Description() string {
	return "8XY0: Sets VX to the value of VY."
}

//

type Or struct{ *RawData }

func (i Or) init() {
	i.s = fmt.Sprintf("OR V%1X,V%1X", i.Register1(), i.Register2())
}
func (i Or) Register1() uint8 { return i.b[0] & 0x0F }
func (i Or) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i Or) Description() string {
	return "8XY1: Sets VX to VX | VY (bit-wise OR)."
}

//

type And struct{ *RawData }

func (i And) init() {
	i.s = fmt.Sprintf("AND V%1X,V%1X", i.Register1(), i.Register2())
}
func (i And) Register1() uint8 { return i.b[0] & 0x0F }
func (i And) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i And) Description() string {
	return "8XY2: Sets VX to VX & VY (bit-wise AND)."
}

//

type Xor struct{ *RawData }

func (i Xor) init() {
	i.s = fmt.Sprintf("XOR V%1X,V%1X", i.Register1(), i.Register2())
}
func (i Xor) Register1() uint8 { return i.b[0] & 0x0F }
func (i Xor) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i Xor) Description() string {
	return "8XY3: Sets VX to VX ^ VY (bit-wise XOR)."
}

//

type AddRegister struct{ *RawData }

func (i AddRegister) init() {
	i.s = fmt.Sprintf("ADD V%1X,V%1X", i.Register1(), i.Register2())
}
func (i AddRegister) Register1() uint8 { return i.b[0] & 0x0F }
func (i AddRegister) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i AddRegister) Description() string {
	return "8XY4: VX += VY. VF = 1 when there's a carry, 0 when there isn't."
}

//

type SubRegister struct{ *RawData }

func (i SubRegister) init() {
	i.s = fmt.Sprintf("SUB V%1X,V%1X", i.Register1(), i.Register2())
}
func (i SubRegister) Register1() uint8 { return i.b[0] & 0x0F }
func (i SubRegister) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i SubRegister) Description() string {
	return "8XY5: VX -= VY. VF = 0 when there's a borrow, 1 when there isn't."
}

//

type Shr struct{ *RawData }

func (i Shr) init() {
	i.s = fmt.Sprintf("SHR V%1X,V%1X", i.Register1(), i.Register2())
}
func (i Shr) Register1() uint8 { return i.b[0] & 0x0F }
func (i Shr) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i Shr) Description() string {
	return "8XY6: VX = VY >> 1. VF = least significant bit prior to the shift."
}

//

type Subn struct{ *RawData }

func (i Subn) init() {
	i.s = fmt.Sprintf("SUBN V%1X,V%1X", i.Register1(), i.Register2())
}
func (i Subn) Register1() uint8 { return i.b[0] & 0x0F }
func (i Subn) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i Subn) Description() string {
	return "8XY7: VX = VY - VX. VF = 0 when there's a borrow, " +
		"1 when there isn't."
}

//

type Shl struct{ *RawData }

func (i Shl) init() {
	i.s = fmt.Sprintf("SHL V%1X,V%1X", i.Register1(), i.Register2())
}
func (i Shl) Register1() uint8 { return i.b[0] & 0x0F }
func (i Shl) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i Shl) Description() string {
	return "8XYE: VX = VY << 1. VF = most significant bit prior to the shift."
}

//

type SneRegister struct{ *RawData }

func (i SneRegister) init() {
	i.s = fmt.Sprintf("SNE V%1X,V%1X", i.Register1(), i.Register2())
}
func (i SneRegister) Register1() uint8 { return i.b[0] & 0x0F }
func (i SneRegister) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i SneRegister) Description() string {
	return "9XY0: Skips the next instruction if VX doesn't equal VY."
}

//

type LdI struct{ *RawData }

func (i LdI) init() {
	i.s = fmt.Sprintf("LD I,%03X", i.Value())
}
func (i LdI) Value() uint16       { return i.Opcode() & 0x0FFF }
func (i LdI) Description() string { return "ANNN: Sets I to the address NNN." }

//

type JpV0 struct{ *RawData }

func (i JpV0) init() {
	i.s = fmt.Sprintf("JP V0,%03X", i.Address())
}
func (i JpV0) Address() uint16 { return i.Opcode() & 0x0FFF }
func (i JpV0) Description() string {
	return "BNNN: Jumps to the address NNN plus V0."
}

//

type Rnd struct{ *RawData }

func (i Rnd) init() {
	i.s = fmt.Sprintf("RND V%1X,%02X", i.Register(), i.Value())
}
func (i Rnd) Register() uint8 { return i.b[0] & 0x0F }
func (i Rnd) Value() uint8    { return i.b[1] }
func (i Rnd) Description() string {
	return "CXNN: Sets VX to a random number (0-FF) & NN (bit-wise AND)."
}

//

type Drw struct{ *RawData }

func (i Drw) init() {
	i.s = fmt.Sprintf("DRW V%1X,V%1X,%1X",
		i.Register1(), i.Register2(), i.Rows())
}
func (i Drw) Register1() uint8 { return i.b[0] & 0x0F }
func (i Drw) Register2() uint8 { return i.b[1] & 0xF0 >> 4 }
func (i Drw) Rows() uint8      { return i.b[1] & 0x0F }
func (i Drw) Description() string {
	return "DXYN: Draws N rows of sprite pointed by I at VX,VY."
}

//

type Skp struct{ *RawData }

func (i Skp) init()           { i.s = fmt.Sprintf("SKP V%1X", i.Register()) }
func (i Skp) Register() uint8 { return i.b[0] & 0x0F }
func (i Skp) Description() string {
	return "EX9E: Skips the next instruction if the " +
		"key stored in VX is pressed."
}

//

type Sknp struct{ *RawData }

func (i Sknp) init()           { i.s = fmt.Sprintf("SKNP V%1X", i.Register()) }
func (i Sknp) Register() uint8 { return i.b[0] & 0x0F }
func (i Sknp) Description() string {
	return "EXA1: Skips the next instruction if the key stored " +
		"in VX isn't pressed."
}

//

type LdDelayTimer struct{ *RawData }

func (i LdDelayTimer) init() {
	i.s = fmt.Sprintf("LD V%1X, DT", i.Register())
}
func (i LdDelayTimer) Register() uint8 { return i.b[0] & 0x0F }
func (i LdDelayTimer) Description() string {
	return "FX07: Sets VX to the value of the delay timer."
}

//

type LdKeyboard struct{ *RawData }

func (i LdKeyboard) init() {
	i.s = fmt.Sprintf("LD V%1X, K", i.Register())
}
func (i LdKeyboard) Register() uint8 { return i.b[0] & 0x0F }
func (i LdKeyboard) Description() string {
	return "FX0A: A key press is awaited, and then key number is stored in VX."
}

//

type LdSetDelayTimer struct{ *RawData }

func (i LdSetDelayTimer) init() {
	i.s = fmt.Sprintf("LD DT,V%1X", i.Register())
}
func (i LdSetDelayTimer) Register() uint8 { return i.b[0] & 0x0F }
func (i LdSetDelayTimer) Description() string {
	return "FX15: Sets the delay timer to VX."
}

//

type LdSetSoundTimer struct{ *RawData }

func (i LdSetSoundTimer) init() {
	i.s = fmt.Sprintf("LD ST,V%1X", i.Register())
}
func (i LdSetSoundTimer) Register() uint8 { return i.b[0] & 0x0F }
func (i LdSetSoundTimer) Description() string {
	return "FX18: Sets the sound timer to VX."
}

//

type AddI struct{ *RawData }

func (i AddI) init() {
	i.s = fmt.Sprintf("ADD I,V%1X", i.Register())
}
func (i AddI) Register() uint8     { return i.b[0] & 0x0F }
func (i AddI) Description() string { return "FX1E: Adds VX to I." }

//

type LdFont struct{ *RawData }

func (i LdFont) init() {
	i.s = fmt.Sprintf("LD I,CHAR V%1X", i.Register())
}
func (i LdFont) Register() uint8 { return i.b[0] & 0x0F }
func (i LdFont) Description() string {
	return "FX29: Sets I to the location of the sprite for the character in VX."
}

//

type LdBcd struct{ *RawData }

func (i LdBcd) init() {
	i.s = fmt.Sprintf("LD [I],BCD V%1X", i.Register())
}
func (i LdBcd) Register() uint8 { return i.b[0] & 0x0F }
func (i LdBcd) Description() string {
	return "FX33: Store BCD representation of VX in memory at I, I+1, and I+2."
}

//

type LdSetMemory struct{ *RawData }

func (i LdSetMemory) init() {
	i.s = fmt.Sprintf("LD [I],V%1X", i.Register())
}
func (i LdSetMemory) Register() uint8 { return i.b[0] & 0x0F }
func (i LdSetMemory) Description() string {
	return "FX55: Stores V0 to VX in memory starting at address I."
}

//

type LdMemory struct{ *RawData }

func (i LdMemory) init() {
	i.s = fmt.Sprintf("LD V%1X,[I]", i.Register())
}
func (i LdMemory) Register() uint8 { return i.b[0] & 0x0F }
func (i LdMemory) Description() string {
	return "FX65: Fills V0 to VX with values from memory starting at address I."
}

// -----------------------------------------------------------------------------

// DisassembleSimple disassembles raw data and return an array of instructions.
// It's fast but it cannot handle odd-aligned opcodes or recognize raw data
// memory regions. For proper disassembly, see Disassemble [to be implemented]
// which runs the program and analyzes it.
func DisassembleSimple(b []byte) (res []Instruction, err error) {
	if len(b)%2 != 0 {
		err = fmt.Errorf("Odd-aligned opcodes are not supported. Please use " +
			"Dissassemble().")
		return
	}

	for i := 0; i < len(b); i += 2 {
		opcode := b[i : i+2]

		rd := &RawData{b: opcode}
		in := Instruction(rd)

		switch opcode[0] & 0xF0 {
		case 0x00:
			in = Sys{rd}
		case 0x10:
			in = Jp{rd}
		case 0x20:
			in = Call{rd}
		case 0x30:
			in = Se{rd}
		case 0x40:
			in = Sne{rd}
		case 0x50:
			in = SeRegister{rd}
		case 0x60:
			in = Ld{rd}
		case 0x70:
			in = Add{rd}
		case 0x80:
			switch opcode[1] & 0x0F {
			case 0x0:
				in = LdRegister{rd}
			case 0x1:
				in = Or{rd}
			case 0x2:
				in = And{rd}
			case 0x3:
				in = Xor{rd}
			case 0x4:
				in = AddRegister{rd}
			case 0x5:
				in = SubRegister{rd}
			case 0x6:
				in = Shr{rd}
			case 0x7:
				in = Subn{rd}
			case 0xE:
				in = Shl{rd}
			}
		case 0x90:
			in = Sne{rd}
		case 0xA0:
			in = LdI{rd}
		case 0xB0:
			in = JpV0{rd}
		case 0xC0:
			in = Rnd{rd}
		case 0xD0:
			in = Drw{rd}
		case 0xE0:
			switch opcode[1] {
			case 0x9E:
				in = Skp{rd}
			case 0xA1:
				in = Sknp{rd}
			}
		case 0xF0:
			switch opcode[1] {
			case 0x07:
				in = LdDelayTimer{rd}
			case 0x0A:
				in = LdKeyboard{rd}
			case 0x15:
				in = LdSetDelayTimer{rd}
			case 0x18:
				in = LdSetSoundTimer{rd}
			case 0x1E:
				in = AddI{rd}
			case 0x29:
				in = LdFont{rd}
			case 0x33:
				in = LdBcd{rd}
			case 0x55:
				in = LdSetMemory{rd}
			case 0x65:
				in = LdMemory{rd}
			}
		}

		in.init()
		res = append(res, in)
	}

	return
}

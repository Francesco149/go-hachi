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

// Package hachi implements various CHIP-8 utilities, including an emulator and
// a disassembler.
package hachi

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"time"
	"unsafe"
)

// -----------------------------------------------------------------------------

// An OutOfMemoryErr is returned upon attempting to load a program that
// exceeds the memory's capacity.
type OutOfMemoryErr struct {
	Instance    *Chip8
	ProgramSize int64
}

func (e *OutOfMemoryErr) Error() string {
	return fmt.Sprintf("Not enough memory (program size: %v, free memory: %v)",
		e.ProgramSize, len(e.Instance.Memory)-0x200)
}

// An StackOverflowErr is returned when the stack pointer exceeds the stack.
type StackOverflowErr struct{}

func (e *StackOverflowErr) Error() string {
	return "Stack overflow."
}

// A BadCodeErr is returned when the emulator tries to execute invalid code.
type BadCodeErr struct{}

func (e *BadCodeErr) Error() string {
	return "Tried to execute invalid code."
}

// A OverflowErr is returned when an overflow occurs during an instruction.
type OverflowErr struct{}

func (e *OverflowErr) Error() string {
	return "Overflow."
}

// A AccessErr is returned when the program tries to access invalid or protected
// memory regions.
type AccessErr struct{}

func (e *AccessErr) Error() string {
	return "Tried to access invalid or protected memory."
}

// -----------------------------------------------------------------------------

// Chip8Settings holds the configuration parameters for a Chip8 instance.
type Chip8Settings struct {
	// Memory size. Max. 0xFFFF (65535).
	MemorySize uint16
	// Stack size. Defines the maximum amount of nested calls.
	StackSize int
	// Screen width and height in pixels. Max. 255x255.
	Width, Height uint8
	// Realistic, when enabled, makes the stack and screen buffers use the
	// same memory regions as the original implementation. This limits the
	// stack to max. 12 levels and the screen buffer to max. 2048 pixels.
	Realistic bool
	// Enables old behaviour for SHL VX,VY , SHR VX,VY , LD [I],VX and LD VX,[I]
	LegacyMode bool
}

// Validate validates the settings.
// Returns an error when the settings aren't valid.
func (s *Chip8Settings) Validate() error {
	if s.Width%8 != 0 {
		return fmt.Errorf("Width must be a multiple of 8, got %v.", s.Width)
	}
	if s.Height%8 != 0 {
		return fmt.Errorf("Height must be a multiple of 8, got %v.", s.Height)
	}
	if s.Width < 8 {
		return fmt.Errorf("Height must be >= 8, got %v.", s.Width)
	}
	if s.Height < 15 {
		return fmt.Errorf("Height must be >= 15, got %v.", s.Height)
	}
	if s.Realistic {
		if s.StackSize > 12 {
			return fmt.Errorf("StackSize must be <= 12 in realistic mode"+
				", got %v.", s.StackSize)
		}
		pixelCount := uint16(s.Width) * uint16(s.Height)
		if pixelCount > 2048 {
			return fmt.Errorf("Width*Height must be <= 2048 in realistic mode"+
				", got %v.", pixelCount)
		}
	}

	return nil
}

// The default settings for Chip8, which mimick the original CHIP-8
// implementation
var DefaultSettings = &Chip8Settings{
	MemorySize: 0x1000,
	StackSize:  12,
	Width:      64, Height: 32,
	Realistic:  true,
	LegacyMode: false,
}

// -----------------------------------------------------------------------------

// Key flags for the Keyboard bitfield.
const (
	Key0 = 1 << iota
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
)

// Key flags mapped by number.
var KeyFlags []uint16 = []uint16{Key0, Key1, Key2, Key3, Key4, Key5, Key6, Key7,
	Key8, Key9, KeyA, KeyB, KeyC, KeyD, KeyE, KeyF}

// Key numbers mapped by flag.
var KeyNumbers map[uint16]uint8 = map[uint16]uint8{
	Key0: 0x00,
	Key1: 0x01,
	Key2: 0x02,
	Key3: 0x03,
	Key4: 0x04,
	Key5: 0x05,
	Key6: 0x06,
	Key7: 0x07,
	Key8: 0x08,
	Key9: 0x09,
	KeyA: 0x0A,
	KeyB: 0x0B,
	KeyC: 0x0C,
	KeyD: 0x0D,
	KeyE: 0x0E,
	KeyF: 0x0F,
}

// -----------------------------------------------------------------------------

// Chip8 is an implementation of a CHIP-8 emulator. It holds the state of the
// virtual machine and provides debugging tools.
type Chip8 struct {
	// The memory where programs are loaded and executed.
	// Most implementations use 4k (0x1000 bytes).
	// Programs normally start at 0x200 because the original interpreter
	// occupied those first 512 bytes.
	Memory []byte
	// V[0x0]~V[0xF] are 8-bit registers. V[0xF] doubles as a carry flag.
	V [16]uint8
	// 16-bit address register. Used for memory operations.
	I uint16
	// The call stack, which holds return addresses.
	// The original implementation allocated 48bytes for up to 12 nested calls.
	// In realistic mode, this is at 0xEA0 through 0xEAC in memory.
	Stack []uint16
	// The stack pointer. Index of the last value that was pushed on stack.
	SP int
	// Program counter. Holds the currently executing address.
	PC uint16
	// Timers. These automatically count down at 60hz when they are non-zero.
	// DT/DelayTimer is intended to be used for timing events in games, while
	// ST/SoundTimer makes a beeping sound as long as its value is non-zero.
	DT uint8
	ST uint8
	// Keyboard is a hex keyboard with 16 keys. 8, 4, 6 and 2 are typically used
	// for directional input.
	// This is a bitfield, see the constants for the flags.
	Keyboard uint16
	// Screen buffer. The official resolution is 64x32.
	// Color is monochrome, so each bit is a pixel which can be either
	// on or off.
	// Because it's stored as an array of bytes, each element holds 8 pixels and
	// the screen size must be a multiple of 8.
	// In realistic mode, this is at 0xF00 through 0xFFF in memory.
	Screen        []byte
	Width, Height uint8
	// The interval between each timer tick. The original implementation uses
	// 60hz = time.Second / 60.
	TimerInterval time.Duration

	lastTimerUpdate time.Time
	driver          string
	wii             *waitInputInfo

	pLdMemory, pLdSetMemory func(c *Chip8, x uint8)
	pShr, pShl              func(c *Chip8, x, y uint8)
}

// -----------------------------------------------------------------------------

// function pointers the legacy mode switch
// (function pointers are a lot faster than if's)

type ldMemoryMap map[bool]func(c *Chip8, x uint8)

var ldMemory = ldMemoryMap{
	false: func(c *Chip8, x uint8) {
		for i := uint8(0); i <= x; i++ {
			c.V[i] = c.Memory[c.I+uint16(i)]
		}
	},
	true: func(c *Chip8, x uint8) {
		for i := uint8(0); i <= x; i++ {
			c.V[i] = c.Memory[c.I]
			c.I++
		}
	},
}

type ldSetMemoryMap map[bool]func(c *Chip8, x uint8)

var ldSetMemory = ldSetMemoryMap{
	false: func(c *Chip8, x uint8) {
		for i := uint8(0); i <= x; i++ {
			c.Memory[c.I+uint16(i)] = c.V[i]
		}
	},
	true: func(c *Chip8, x uint8) {
		for i := uint8(0); i <= x; i++ {
			c.Memory[c.I] = c.V[i]
			c.I++
		}
	},
}

type shlMap map[bool]func(c *Chip8, x, y uint8)

var shl = shlMap{
	false: func(c *Chip8, x, y uint8) {
		c.V[0xF] = c.V[x] & 0x80 // most significant bit
		c.V[x] <<= 1
	},
	true: func(c *Chip8, x, y uint8) {
		c.V[0xF] = c.V[y] & 0x80 // most significant bit
		c.V[x] = c.V[y] << 1
	},
}

type shrMap map[bool]func(c *Chip8, x, y uint8)

var shr = shrMap{
	false: func(c *Chip8, x, y uint8) {
		c.V[0xF] = c.V[x] & 0x01 // least significant bit
		c.V[x] >>= 1
	},
	true: func(c *Chip8, x, y uint8) {
		c.V[0xF] = c.V[y] & 0x01 // least significant bit
		c.V[x] = c.V[y] >> 1
	},
}

// -----------------------------------------------------------------------------

// struct used to hold some info when waiting for input
type waitInputInfo struct {
	register uint8
	zeroBits uint16
}

// New initializes a new instance of Chip8 with the given settings. If settings
// is nil, DefaultSettings will be used.
// driver is the name of the syscall driver that will be used.
func New(driver string, s *Chip8Settings) (c *Chip8, err error) {
	if drivers[driver] == nil {
		err = fmt.Errorf("Driver %s not found.", c.driver)
		return
	}

	if s == nil {
		s = DefaultSettings
	}

	err = s.Validate()
	if err != nil {
		return
	}

	c = &Chip8{
		Memory: make([]uint8, s.MemorySize),
		Width:  s.Width, Height: s.Height,
		TimerInterval: time.Second / 60,
		driver:        driver,
		SP:            -1,
		pLdMemory:     ldMemory[s.LegacyMode],
		pLdSetMemory:  ldSetMemory[s.LegacyMode],
		pShr:          shr[s.LegacyMode],
		pShl:          shl[s.LegacyMode],
	}

	// init realistic mode
	if s.Realistic {
		// ugly slice hack:
		// make Stack point to an area of memory and interpret it as uint16's
		stackmem := c.Memory[0xEA0 : 0xEA0+uint16(s.StackSize)]
		header := *(*reflect.SliceHeader)(unsafe.Pointer(&stackmem))
		cbuint16 := int(unsafe.Sizeof(uint16(0)) / unsafe.Sizeof(byte(0)))
		header.Len /= cbuint16
		header.Cap /= cbuint16
		c.Stack = *(*[]uint16)(unsafe.Pointer(&header))

		c.Screen = c.Memory[0xF00 : 0xF00+uint16(s.Width)*uint16(s.Height)/8]
	} else {
		c.Stack = make([]uint16, s.StackSize)
		c.Screen = make([]uint8, uint16(s.Width)*uint16(s.Height)/8)
	}

	// init fonts
	copy(c.Memory, []byte{
		0xF0, 0x90, 0x90, 0x90, 0xF0,
		0x20, 0x60, 0x20, 0x20, 0x70,
		0xF0, 0x10, 0xF0, 0x80, 0xF0,
		0xF0, 0x10, 0xF0, 0x10, 0xF0,
		0x90, 0x90, 0xF0, 0x10, 0x10,
		0xF0, 0x80, 0xF0, 0x10, 0xF0,
		0xF0, 0x80, 0xF0, 0x90, 0xF0,
		0xF0, 0x10, 0x20, 0x40, 0x40,
		0xF0, 0x90, 0xF0, 0x90, 0xF0,
		0xF0, 0x90, 0xF0, 0x10, 0xF0,
		0xF0, 0x90, 0xF0, 0x90, 0x90,
		0xE0, 0x90, 0xE0, 0x90, 0xE0,
		0xF0, 0x80, 0x80, 0x80, 0xF0,
		0xE0, 0x90, 0x90, 0x90, 0xE0,
		0xF0, 0x80, 0xF0, 0x80, 0xF0,
		0xF0, 0x80, 0xF0, 0x80, 0x80,
	})

	drivers[c.driver].OnInit(c)
	log.Println(c)
	return
}

// String returns formatted information about the instance of the emulator.
func (c *Chip8) String() string {
	return fmt.Sprintf("Chip8{Memory: %v bytes, Registers: [% 02X] I: %04X, "+
		"Stack: % 04X, SP: %v, PC: %04X, DT: %02X, ST: %02X, "+
		"Keyboard: %016b, Screen: %v*%v}",
		len(c.Memory), c.V, c.I, c.Stack[0:c.SP], c.SP, c.PC, c.DT,
		c.ST, c.Keyboard, c.Width, c.Height)
}

// Driver returns the name of the syscall driver in use by the emulator.
func (c *Chip8) Driver() string { return c.driver }

// GetDriverData gets custom data from the currently loaded driver.
// Returns nil if the driver does not exist or if the data key is not found.
func (c *Chip8) GetDriverData(key string) interface{} {
	if drivers[c.driver] == nil {
		return nil
	}
	return drivers[c.driver].GetData(key)
}

// Load opens a CHIP-8 binary file and loads it into memory.
// Returns the size, in bytes, of the program and an error if any.
func (c *Chip8) Load(path string) (size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return
	}

	size = fi.Size()
	if fi.Size() > int64(len(c.Memory)-0x200) {
		err = &OutOfMemoryErr{c, fi.Size()}
		return
	}

	_, err = f.Read(c.Memory[0x200:])
	c.PC = 0x200
	log.Printf(`Loaded %v bytes of code from "%s"`, fi.Size(), path)
	return
}

// LoadRaw loads a byte array as a CHIP-8 binary into memory.
func (c *Chip8) LoadRaw(program []byte) error {
	if len(program) > len(c.Memory)-0x200 {
		return &OutOfMemoryErr{c, int64(len(program))}
	}
	copy(c.Memory[0x200:], program)
	log.Println("Loaded", len(program), "bytes of code")
	return nil
}

// Tick runs one CPU cycle, blocking the thread. Returns an error if any.
func (c *Chip8) Tick() error {
	drivers[c.driver].OnUpdate(c)
	if c.wii != nil {
		changed := c.Keyboard & c.wii.zeroBits
		if changed == 0 {
			return nil
		}

		// get first pressed key (in case multiple are pressed0
		for mask := uint16(0x0001); ; mask <<= 1 {
			c.V[c.wii.register] = KeyNumbers[changed&mask]
			if c.V[c.wii.register] != 0 {
				break
			}
			if mask == 0x8000 {
				break
			}
		}

		c.wii = nil
	}

	opcode := c.Memory[c.PC : c.PC+2]
	c.PC += 2

	// this has lots of code redundancy in favor of speed

	switch opcode[0] & 0xF0 {
	case 0x00:
		// SYS NNN
		// Performs a syscall of the function at address NNN.
		// Since this is an emulator, we're just going to implement E0 and EE,
		// which are CLS and RET.
		// todo: write CLS and RET in CHIP-8 assembly and allocate them in
		//       memory for realism.
		switch uint16(opcode[0]&0x0F)<<8 | uint16(opcode[1]) {
		case 0x0E0: // CLS
			for i := 0; i < len(c.Screen); i++ {
				c.Screen[i] = 0
			}
			drivers[c.driver].Cls()
		case 0x0EE: // RET
			// pop return address
			if c.SP < 0 {
				return &StackOverflowErr{}
			}
			c.PC = c.Stack[c.SP]
			c.SP--
		}
	case 0x10:
		// JP NNN
		c.PC = uint16(opcode[0]&0x0F)<<8 | uint16(opcode[1])
	case 0x20:
		// CALL NNN
		if c.SP >= len(c.Stack)-1 {
			return &StackOverflowErr{}
		}
		// push return address
		c.SP++
		c.Stack[c.SP] = c.PC
		c.PC = uint16(opcode[0]&0x0F)<<8 | uint16(opcode[1])
	case 0x30:
		// SE VX,NN
		if c.V[opcode[0]&0x0F] == opcode[1] {
			c.PC += 2
		}
	case 0x40:
		// SNE VX,NN
		if c.V[opcode[0]&0x0F] != opcode[1] {
			c.PC += 2
		}
	case 0x50:
		// SE VX,VY
		if c.V[opcode[0]&0x0F] == c.V[opcode[1]&0xF0>>4] {
			c.PC += 2
		}
	case 0x60:
		// LD VX,NN
		c.V[opcode[0]&0x0F] = opcode[1]
	case 0x70:
		// ADD VX,NN
		c.V[opcode[0]&0x0F] += opcode[1]
	case 0x80:
		switch opcode[1] & 0x0F {
		case 0x0:
			// LD VX,VY
			c.V[opcode[0]&0x0F] = c.V[opcode[1]&0xF0>>4]
		case 0x1:
			// OR VX,VY
			c.V[opcode[0]&0x0F] |= c.V[opcode[1]&0xF0>>4]
		case 0x2:
			// AND VX,VY
			c.V[opcode[0]&0x0F] &= c.V[opcode[1]&0xF0>>4]
		case 0x3:
			// XOR VX,VY
			c.V[opcode[0]&0x0F] ^= c.V[opcode[1]&0xF0>>4]
		case 0x4:
			// ADD VX,VY
			reg := opcode[0] & 0x0F
			result := uint16(c.V[reg]) +
				uint16(c.V[opcode[1]&0xF0>>4])

			// only store the 8 least significant bits
			c.V[reg] = uint8(result)

			// carry flag
			if result&0xFF00 != 0 {
				c.V[0xF] = 1
			} else {
				c.V[0xF] = 0
			}
		case 0x5:
			// SUB VX,VY
			x := opcode[0] & 0x0F
			y := opcode[1] & 0xF0 >> 4

			// borrow
			if c.V[x] >= c.V[y] {
				c.V[0xF] = 1
			} else {
				c.V[0xF] = 0
			}
			c.V[x] -= c.V[y]

		case 0x6:
			// SHR VX,VY (VX = VY >> 1 or VX >>= 1 in newer implementations)
			c.pShr(c, opcode[0]&0x0F, opcode[1]&0xF0>>4)
		case 0x7:
			// SUBN VX,VY
			x := opcode[0] & 0x0F
			y := opcode[1] & 0xF0 >> 4

			// borrow
			if c.V[x] > c.V[y] {
				c.V[0xF] = 0
			} else {
				c.V[0xF] = 1
			}
			c.V[x] = c.V[y] - c.V[x]
		case 0xE:
			// SHL VX,VY (VX = VY << 1 or VX <<= 1 in newer implementations)
			c.pShl(c, opcode[0]&0x0F, opcode[1]&0xF0>>4)
		default:
			return &BadCodeErr{}
		}
	case 0x90:
		// SNE VX,VY
		if c.V[opcode[0]&0x0F] != c.V[opcode[1]&0xF0>>4] {
			c.PC += 2
		}
	case 0xA0:
		// LD I,NNN
		c.I = uint16(opcode[0]&0x0F)<<8 | uint16(opcode[1])
	case 0xB0:
		// JP V0,NNN
		c.PC = uint16(opcode[0]&0x0F)<<8 | uint16(opcode[1]) +
			uint16(c.V[0]) - 2
	case 0xC0:
		// RND VX,NN (VX = rand() & NN)
		c.V[opcode[0]&0x0F] = uint8(rand.Uint32()) & opcode[1]
	case 0xD0:
		// DRW VX,VY,N
		x := c.V[opcode[0]&0x0F] % c.Width
		y := c.V[opcode[1]&0xF0>>4] % c.Height
		// we have to modulo everything by width and height, that's how
		// the chip-8 handles drawing.

		rows := opcode[1] & 0x0F
		if 0xFFFF-c.I < uint16(rows) {
			return &OverflowErr{}
		}

		if int(c.I)+int(rows)-1 >= len(c.Memory) {
			return &AccessErr{}
		}

		/*
				Screen memory layout (this is the one I implemented):
			                                     x ->
				  00000000 00000000 00000000 00000000
				  00000000 01000000 00000000 00000000
				y 00000000 00000000 00000000 00000000
				| 00000000 00000000 00000000 00000000
				v ...

				the 1 is at screen coordinates 9, 1 but because we are packing
				the screen as single bits in an array of bytes, the 1 is the 2nd
				bit of the 6th element in the byte array (or row 2, column 2
				element if it was a 2D array).

				Essentially, the X coordinate for accessing bytes must be
				divided by 8, and then we must shift our bitmask by the
				remainder.

				To flip the bit, we would need to:

				x := 9
				y := 1

				byteIndex := y*width/8 + x/8
				// 1*32/8 + 9/8 = 5

				bitOffset := x%8
				// 9%8 = 1

				mask := 0x80>>bitOffset
				// 0b10000000>>1 = 0b01000000

				screen[index] ^= mask

				----------------------------------------------------------------

				Screen memory layout (alternative, not sure which one the real
				thing actually uses, but it's most likely the previous one):
			                   y ->
				  00000000 00000000
				  00000000 00000000
				  00000000 00000000
				  00000000 00000000
				  00000000 00000000
				  00000000 00000000
				  00000000 00000000
				  00000000 00000000
				x 00000000 00000000
				| 01000000 00000000
				v ...

				the 1 is at screen scoordinates 9, 1 but because we are packing
				the screen as single bits in an array of bytes, the 1 is in the
				2nd bit of the 19th element in the byte array, so it's actually
				at byte 9,0.

				Essentially, the Y coordinate for accessing bytes must be
				divided by 8, and then we must shift our bitmask by the
				remainder.

				To flip the bit, we would need to:

				x := 9
				y := 1

				byteIndex := x*height/8 + y/8
				// 9*16/8 + 1/8 = 18

				bitOffset := y%8
				// 1%8 = 1

				mask := 0x80>>bitOffset
				// 0b10000000>>1 = 0b01000000

				screen[index] ^= mask

				note that sprite's bytes will need to be shifted bit by bit and
				xored with the bitoff-th bit of each column byte
		*/

		c.V[0xF] = 0
		sprite := c.Memory[c.I : c.I+uint16(rows)]

		byteWidth := uint16(c.Width) / 8

		for off := uint8(0); off < rows; off++ {
			// index in the screen byte array
			byteColumn := uint16(y) * byteWidth
			index := byteColumn + uint16(x)/8
			nextIndex := byteColumn + (uint16(x)/8+1)%byteWidth
			// make sure we modulo the next X for the wrap-around behaviour

			// start xoring at bitoff bits
			bitoff := x % 8

			// mask for current byte and next byte
			mask1 := uint8(0xFF) >> bitoff
			mask2 := ^mask1

			// store old vals, ignoring the bits we don't use
			oldval1 := c.Screen[index] & mask1
			c.Screen[index] ^= sprite[off] >> bitoff

			var oldval2 byte
			if bitoff != 0 {
				oldval2 = c.Screen[nextIndex] & mask2
				c.Screen[nextIndex] ^= sprite[off] << (8 - bitoff)
			}

			// set VF to 1 if any pixels were cleared (collision)
			for mask := uint8(0x01); c.V[0xF] == 0; mask <<= 1 {
				if oldval1&mask > c.Screen[index]&mask1&mask {
					// previous bit was set and it's now unset, which means
					// that we have a collision
					c.V[0xF] = 1
					break
				}
				if bitoff != 0 &&
					oldval2&mask > c.Screen[nextIndex]&mask2&mask {
					// same as above
					c.V[0xF] = 1
					break
				}
				if mask == 0x80 {
					break
				}
			}

			y = (y + 1) % c.Height // don't forget to modulo
		}

		drivers[c.driver].UpdateScreen(c)
	case 0xE0:
		switch opcode[1] {
		case 0x9E:
			// SKP VX
			if c.Keyboard&KeyFlags[c.V[opcode[0]&0x0F]] != 0 {
				c.PC += 2
			}
		case 0xA1:
			// SKNP VX
			if c.Keyboard&KeyFlags[c.V[opcode[0]&0x0F]] == 0 {
				c.PC += 2
			}
		default:
			return &BadCodeErr{}
		}
	case 0xF0:
		switch opcode[1] {
		case 0x07:
			// LD VX,DT
			c.V[opcode[0]&0x0F] = c.DT
		case 0x0A:
			// LD VX,K
			// wait for input
			c.wii = &waitInputInfo{opcode[0] & 0x0F, ^c.Keyboard}
		case 0x15:
			// LD DT,VX
			c.DT = c.V[opcode[0]&0x0F]
		case 0x18:
			// LD ST,VX
			c.ST = c.V[opcode[0]&0x0F]
		case 0x1E:
			// ADD I,VX
			vx := uint16(c.V[opcode[0]&0x0F])
			if vx > 0xFFFF-c.I {
				// undocumented feature - set VF to 1 when there's a
				// range overflow.
				//c.V[0xF] = 1
			} else {
				//c.V[0xF] = 0
			}
			c.I += vx
		case 0x29:
			// LD LD I,CHAR VX
			// fonts are stored starting at 0x0000
			c.I = uint16(c.V[opcode[0]&0x0F]) * 5
		case 0x33:
			// LD [I],BCD VX
			if int(c.I)+2 >= len(c.Memory) || c.I < 0x200 {
				return &AccessErr{}
			}
			value := c.V[opcode[0]&0x0F]
			c.Memory[c.I+2] = value % 10 // ones
			value /= 10
			c.Memory[c.I+1] = value % 10 // tens
			c.Memory[c.I] = value / 10   // hundreds

		case 0x55:
			// LD [I],VX
			x := opcode[0] & 0x0F

			// check for overflow
			if 0xFFFF-c.I < uint16(x) {
				return &OverflowErr{}
			}

			// check for out of bounds memory
			if int(c.I)+int(x) >= len(c.Memory) || c.I < 0x200 {
				return &AccessErr{}
			}

			// copy memory to V0-VX
			c.pLdSetMemory(c, x)
		case 0x65:
			// LD VX,[I]
			x := opcode[0] & 0x0F

			// check for overflow
			if 0xFFFF-c.I < uint16(x) {
				return &OverflowErr{}
			}

			// check for out of bounds memory
			if int(c.I)+int(x) >= len(c.Memory) || c.I < 0x200 {
				return &AccessErr{}
			}

			// copy memory from V0-VX
			c.pLdMemory(c, x)
		default:
			return &BadCodeErr{}
		}
	default:
		return &BadCodeErr{}
	}

	now := time.Now()

	if c.lastTimerUpdate.IsZero() {
		c.lastTimerUpdate = now
	}

	for now.Sub(c.lastTimerUpdate) >= c.TimerInterval {
		if c.DT > 0 {
			c.DT--
		}
		if c.ST > 0 {
			c.ST--
			drivers[c.driver].Beep()
		}
		c.lastTimerUpdate = c.lastTimerUpdate.Add(c.TimerInterval)
	}

	return nil
}

// Run runs the emulator, blocking the thread.
// Exits and returns an error if any.
func (c *Chip8) Run() (err error) {
	for err == nil {
		err = c.Tick()
	}
	return
}

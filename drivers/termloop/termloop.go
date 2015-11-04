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

// Package termloop implements a syscall driver for termloop.
//
// The driver initializes a termloop context which can then be retrieved from
// GetDriverData("ctx"). The caller must then set up an entity that calls
// Tick() on the emulator instance on every Draw call.
//
// Key mappings can be modified through SetDriverData("key_map", myMap), where
// myMap is a map map[termloop.Key]uint16 with termloop keys as keys and
// Chip-8 keys (hachi.Key0...hachi.KeyF) as values.
package termloop

import (
	"fmt"
	"github.com/Francesco149/go-hachi/hachi"
	tl "github.com/JoelOtter/termloop"
	"log"
	"reflect"
	"time"
)

// A TermloopDriver is a terminal-based driver that uses the termloop library.
// It shows the current emulator state in real time and the screen.
type TermloopDriver struct {
	hachi.Driver
	g                 *tl.Game
	memory            *tl.Text
	registers         *tl.Text
	pointersAndTimers *tl.Text
	devices           *tl.Text
	stack             []*tl.Text
	syscalls          [10]*tl.Text
	screen            [][]*tl.Rectangle
	lastScreen        []byte
	keyMap            map[tl.Key]uint16
}

func (d *TermloopDriver) printSyscall(s string) {
	for i := 1; i < 10; i++ {
		d.syscalls[i].SetText(d.syscalls[i-1].Text())
	}
	d.syscalls[0].SetText(s)
}

// just a wrapper entity to handle input
type inputHandler struct {
	c      *hachi.Chip8
	d      *TermloopDriver
	timers map[uint16]time.Time
	// since termbox only polls for keydown events we need to add a timer to
	// automatically release those keys
}

func (i *inputHandler) Draw(s *tl.Screen) {
	for key, t := range i.timers {
		if i.c.Keyboard&key == 0 {
			continue
		}
		if time.Since(t) > time.Millisecond*100 {
			i.c.Keyboard &= ^key
		}
	}
}

func (i *inputHandler) Tick(ev tl.Event) {
	if ev.Type == tl.EventKey {
		keyMask := i.d.keyMap[ev.Key]
		i.c.Keyboard |= keyMask
		i.timers[keyMask] = time.Now()
	}
}

func (d *TermloopDriver) OnInit(c *hachi.Chip8) {
	// hex keyboard with 16 keys.
	// 8, 4, 6 and 2 are typically used for directional input.
	d.keyMap = map[tl.Key]uint16{
		tl.KeyTab:        hachi.Key0,
		tl.KeyF2:         hachi.Key1,
		tl.KeyF3:         hachi.Key2,
		tl.KeyF4:         hachi.Key3,
		tl.KeyF5:         hachi.Key4,
		tl.KeyF6:         hachi.Key5,
		tl.KeyF7:         hachi.Key6,
		tl.KeyF8:         hachi.Key7,
		tl.KeyF9:         hachi.Key8,
		tl.KeyF10:        hachi.Key9,
		tl.KeyCtrlA:      hachi.KeyA,
		tl.KeyCtrlB:      hachi.KeyB,
		tl.KeyCtrlC:      hachi.KeyC,
		tl.KeyCtrlD:      hachi.KeyD,
		tl.KeyCtrlE:      hachi.KeyE,
		tl.KeyCtrlF:      hachi.KeyF,
		tl.KeyArrowDown:  hachi.Key2,
		tl.KeyArrowLeft:  hachi.Key4,
		tl.KeyArrowRight: hachi.Key6,
		tl.KeyArrowUp:    hachi.Key8,
		tl.KeyEnter:      hachi.Key5,
	}

	// init termloop
	d.g = tl.NewGame()
	scr := d.g.Screen()

	scr.AddEntity(&inputHandler{c, d, make(map[uint16]time.Time)})
	scr.AddEntity(tl.NewText(0, 0, "Stack   Syscalls",
		tl.ColorDefault, tl.ColorDefault))

	// stack
	d.stack = make([]*tl.Text, len(c.Stack))
	for i := 0; i < len(d.stack); i++ {
		d.stack[i] = tl.NewText(
			0, i+1, "", tl.ColorDefault, tl.ColorDefault)
		d.g.Screen().AddEntity(d.stack[i])
	}

	// syscall log
	for i := 0; i < 10; i++ {
		d.syscalls[i] = tl.NewText(
			8, i+1, "", tl.ColorDefault, tl.ColorDefault)
		d.g.Screen().AddEntity(d.syscalls[i])
	}

	// chip info
	d.memory = tl.NewText(20, 0, "placeholder",
		tl.ColorDefault, tl.ColorDefault)
	scr.AddEntity(d.memory)

	d.registers = tl.NewText(20, 1, "placeholder",
		tl.ColorDefault, tl.ColorDefault)
	scr.AddEntity(d.registers)

	d.pointersAndTimers = tl.NewText(20, 2, "placeholder",
		tl.ColorDefault, tl.ColorDefault)
	scr.AddEntity(d.pointersAndTimers)

	d.devices = tl.NewText(20, 3, "placeholder",
		tl.ColorDefault, tl.ColorDefault)
	scr.AddEntity(d.devices)

	// screen preview at 20,5
	d.screen = make([][]*tl.Rectangle, c.Width)
	color := tl.ColorWhite // foreground

	for i := uint8(0); i < c.Width; i++ {
		d.screen[i] = make([]*tl.Rectangle, c.Height)

		for j := uint8(0); j < c.Height; j++ {
			d.screen[i][j] = tl.NewRectangle(
				20+int(i), 5+int(j),
				1, 1, color,
			)
		}
	}

	d.lastScreen = make([]byte, uint16(c.Width)*uint16(c.Height)/8)
	log.Println("TermloopDriver initialized")
}

func (d *TermloopDriver) cls() {
	scr := d.g.Screen()
	for i := 0; i < len(d.screen); i++ {
		for j := 0; j < len(d.screen[i]); j++ {
			scr.RemoveEntity(d.screen[i][j])
		}
	}
}

func (d *TermloopDriver) Cls() {
	d.printSyscall("CLS")
	//d.cls()
	// removed because it causes graphical glitches
}

func (d *TermloopDriver) OnUpdate(c *hachi.Chip8) {
	// update chip info
	d.memory.SetText(fmt.Sprintf("Memory: %v bytes", len(c.Memory)))
	d.registers.SetText(fmt.Sprintf("Registers: % 02X", c.V))
	d.pointersAndTimers.SetText(
		fmt.Sprintf("I: %04X SP: %v, PC: %04X, DT: %02X, ST: %02X",
			c.I, c.SP, c.PC, c.DT, c.ST))

	d.devices.SetText(fmt.Sprintf("Keyboard: %016b, Screen: %v*%v",
		c.Keyboard, c.Width, c.Height))

	// update stack
	for i := 0; i < len(c.Stack); i++ {
		if i <= c.SP {
			d.stack[i].SetText(fmt.Sprintf("%04X", c.Stack[i]))
		} else {
			d.stack[i].SetText("")
		}
	}
}

func (d *TermloopDriver) UpdateScreen(c *hachi.Chip8) {
	d.printSyscall("DRW")

	if len(c.Screen) != len(d.lastScreen) {
		// this should handle unlikely resolution changes at runtime
		d.cls()
		d.screen = make([][]*tl.Rectangle, c.Width)
		d.lastScreen = make([]byte, uint16(c.Width)*uint16(c.Height)/8)
	}

	scr := d.g.Screen()
	byteWidth := c.Width / 8
	for i := uint8(0); i < byteWidth; i++ {
		for j := uint8(0); j < c.Height; j++ {
			// index in the screen byte array
			index := uint16(j)*uint16(byteWidth) + uint16(i)

			b1 := d.lastScreen[index]
			b2 := c.Screen[index]

			// iterate this group of 8 pixels/bits and see what changed
			mask := uint8(0x80)
			for bit := uint8(0); bit < 8; bit++ {
				if b2&mask > b1&mask {
					// this pixel was activated
					scr.AddEntity(d.screen[i*8+bit][j])
				} else if b2&mask < b1&mask {
					// this pixel was deactivated
					scr.RemoveEntity(d.screen[i*8+bit][j])
				}
				mask >>= 1
			}
		}
	}

	copy(d.lastScreen, c.Screen)
}

func (d *TermloopDriver) Beep() { d.printSyscall("BEEP") }

func (d *TermloopDriver) GetData(key string) interface{} {
	if key == "ctx" {
		return d.g
	}
	return nil
}

func (d *TermloopDriver) SetData(key string, value interface{}) error {
	if key == "key_map" {
		newMap, ok := value.(map[tl.Key]uint16)
		if !ok {
			fmt.Errorf("Invalid type %s for key_map.", reflect.TypeOf(value))
		}
		d.keyMap = newMap
	}
	return fmt.Errorf("Unknown data key '%s'.", key)
}

// -----------------------------------------------------------------------------

func init() {
	err := hachi.RegisterDriver("termloop", &TermloopDriver{})
	if err != nil {
		log.Fatal(err)
	}
}

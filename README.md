go-hachi is a CHIP-8 emulator and disassembler written in Go. It is designed to 
be modular and pluggable into any front-end by simply implementing a syscall 
driver.

Currently, the emulator works for most standard CHIP-8 programs, but there are
some bugs with very old programs, probably due to undocumented or legacy 
features that I'm not aware of. I'm also planning on adding super chip support.

I'm having a blast coding this and it's a great first emulation project.

Usage
================================================================================
First of all, get and build package hachi (the core):
```
go get github.com/Francesco149/go-hachi/hachi
go install github.com/Francesco149/go-hachi/hachi
```

Now you can build your desired front-end and associated driver. For now, the 
only available front-end is termloop.
```
go get github.com/Francesco149/go-hachi/drivers
go install github.com/Francesco149/go-hachi/drivers/termloop
go get github.com/Francesco149/go-hachi/tl-hachi
go install github.com/Francesco149/go-hachi/tl-hachi
```

Running the emulator is as easy as:
```
cd $GOPATH/bin
tl-hachi /path/to/program.ch8
```

For the default key bindings, check the driver's source file.
The default ones for the termloop driver are:
```go
{
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
```

Implementing your own driver
================================================================================
```go
package mydriver

import (
	"github.com/Francesco149/go-hachi/hachi"
	"log"
	"fmt"
)

type MyDriver struct {
	hachi.Driver
	// other fields
}

func (d *MyDriver) OnInit(c *hachi.Chip8) {
	// do init stuff
	log.Println("MyDriver initialized")
}

func (d *MyDriver) Cls() {
	// handle clear-screen call
	// NOTE: it's not recommended to actually clear the screen buffer here, as
	// it can cause glitches.
}

func (d *MyDriver) OnUpdate(c *hachi.Chip8) {
	// handle input and do other update logic
}

func (d *MyDriver) UpdateScreen(c *hachi.Chip8) {
	// handle draw call
}

func (d *MyDriver) Beep() { 
	// handle beep call
}

func (d *MyDriver) GetData(key string) interface{} {
	switch key {
		// ... (return any custom data your driver might require)
	}
	return nil
}

func (d *MyDriver) SetData(key string, value interface{}) error {
	switch key {
		// ... (set any custom data your driver might require)
	}
	return fmt.Errorf("Unknown data key '%s'.", key)
}

// -----------------------------------------------------------------------------

func init() {
	// register your driver
	err := hachi.RegisterDriver("mydriver", &MyDriver{})
	if err != nil {
		log.Fatal(err)
	}
}
```

Using a driver
================================================================================
```go
package main

import (
	_ "path/to/mydriver"
	"github.com/Francesco149/go-hachi/hachi"
	"log"
	"os"
	"path/filepath"
)

func runEmulator(file string) (err error) {
	// initialize emulator
	ha, err := hachi.New("mydriver", nil)
	if err != nil {
		return
	}

	// load program
	_, err := ha.Load(file)
	if err != nil {
		return
	}

	return ha.Run()
}

func main() {
	log.SetOutput(os.Stdout)
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s path/to/program", filepath.Base(os.Args[0]))
		return
	}
	err := runEmulator(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
}
```

Using the disassembler
================================================================================
Note that the disassembler only works for simple non-odd-aligned programs for 
now. I will eventually write a more advanced one.
```go
package main

import (
	_ "path/to/mydriver"
	"github.com/Francesco149/go-hachi/hachi"
	"log"
	"os"
	"fmt"
	"path/filepath"
)

func runDisassembler(file string) (err error) {
	// initialize emulator
	ha, err := hachi.New("mydriver", nil)
	if err != nil {
		return
	}

	// load program
	progSize, err := ha.Load(file)
	if err != nil {
		return
	}

	// disassemble
	disassembly, err := hachi.DisassembleSimple(
		ha.Memory[0x200 : 0x200+progSize])
	if err != nil {
		return
	}

	// print disassembly
	for _, opcode := range disassembly {
		fmt.Println(opcode)
	}
}

func main() {
	log.SetOutput(os.Stdout)
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s path/to/program", filepath.Base(os.Args[0]))
		return
	}
	err := runDisassembler(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
}
```

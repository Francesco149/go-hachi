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

package main

import (
	"fmt"
	_ "github.com/Francesco149/go-hachi/drivers/termloop"
	"github.com/Francesco149/go-hachi/hachi"
	tl "github.com/JoelOtter/termloop"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"
)

// just a wrapper entity to call the emulator's tick function on every frame
type emulatorWrapper struct{ ha *hachi.Chip8 }

func (e *emulatorWrapper) Draw(s *tl.Screen) {
	err := e.ha.Tick()
	if err != nil {
		log.Println(e.ha)
		log.Fatal(err)
	}
	// we must use Draw because Tick is only called on input
}
func (e *emulatorWrapper) Tick(ev tl.Event) {}

func runEmulator(file string) (err error) {
	// initialize emulator
	ha, err := hachi.New("termloop", nil)
	if err != nil {
		return
	}

	// load program
	progSize, err := ha.Load(file)
	if err != nil {
		return
	}

	// initialize termloop
	ctx := ha.GetDriverData("ctx")
	g, ok := ctx.(*tl.Game)
	if !ok {
		return fmt.Errorf("Driver context failed type assertion.")
	}
	if g == nil {
		return fmt.Errorf("Driver context is nil.")
	}

	// add emulator entity
	g.Screen().AddEntity(&emulatorWrapper{ha})

	// start termloop
	g.Start()

	// -------

	disassembly, err := hachi.DisassembleSimple(
		ha.Memory[0x200 : 0x200+progSize])
	if err != nil {
		return
	}

	w := new(tabwriter.Writer)

	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	fmt.Fprintln(w, "addr\topcode\tpseudo-code\tascii\tdescription\t")

	address := 0x200
	for _, i := range disassembly {
		asciitext := ""
		ascii := i.ASCII()
		if len(ascii) != 0 {
			asciitext = fmt.Sprintf("`%s`", ascii)
		}

		opcodeFormatter := "%04X"
		if i.Size() == 1 {
			opcodeFormatter = "%02X"
		}

		fmt.Fprintf(w, "%04X\t"+opcodeFormatter+"\t%v\t%s\t%s\n",
			address, i.Opcode(), i, asciitext, i.Description())

		address += i.Size()
	}

	w.Flush()
	return
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

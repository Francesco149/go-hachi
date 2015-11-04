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

import (
	"fmt"
	"log"
)

// A Driver is an interface through which the emulator can perform plarform
// specific calls.
// Drivers should be registered by the RegisterDriver function in init().
type Driver interface {
	// Called before the emulator starts executing the program.
	OnInit(c *Chip8)
	// Clears the screen.
	Cls()
	// Called on every clock cycle, should be used for input polling and similar
	// tasks.
	OnUpdate(c *Chip8)
	// Called when the program modifies the screen buffer.
	UpdateScreen(c *Chip8)
	// Plays a beeping sound (this will be called every 1/60th of a second)
	Beep()
	// Returns custom data that can be retrieved through the emulator by
	// calling GetDriverData()
	GetData(key string) interface{}
	// Sets custom data that can be set through the emulator by
	// calling SetDriverData()
	SetData(key string, value interface{}) error
}

// -----------------------------------------------------------------------------

var drivers map[string]Driver

// RegisterDriver registers a driver to a name. The driver can then be used
// by setting the Driver field of Chip8 to the driver's name.
// This is not thread-safe, so don't call it concurrently to the emulator's
// execution.
func RegisterDriver(name string, drv Driver) error {
	if drivers[name] != nil {
		return fmt.Errorf("Driver %s already exists.", name)
	}
	drivers[name] = drv
	return nil
}

// UnregisterDriver unloads a previously registered driver.
// This is not thread-safe, so don't call it concurrently to the emulator's
// execution.
func UnregisterDriver(name string) error {
	if drivers[name] == nil {
		return fmt.Errorf("Driver %s does not exists.", name)
	}
	delete(drivers, name)
	return nil
}

// -----------------------------------------------------------------------------

// A NullDriver is the default driver, which ignores all calls.
type NullDriver struct{ Driver }

func (d NullDriver) OnInit(c *Chip8)                {}
func (d NullDriver) Cls()                           {}
func (d NullDriver) OnUpdate(c *Chip8)              {}
func (d NullDriver) UpdateScreen(c *Chip8)          {}
func (d NullDriver) Beep()                          {}
func (d NullDriver) GetData(key string) interface{} { return nil }
func (d NullDriver) SetData(key string, value interface{}) error {
	return fmt.Errorf("This driver has no settable data.")
}

// -----------------------------------------------------------------------------

func init() {
	drivers = make(map[string]Driver)

	err := RegisterDriver("null", &NullDriver{})
	if err != nil {
		log.Fatal(err)
	}
}

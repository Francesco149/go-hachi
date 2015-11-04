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

// Package drivers contains various syscall drivers for hachi.
// Importing this package loads and registers all drivers. If you only need to
// use one of them (which is usually the case), just import the specific driver
// package.
// To implement your own drivers, see the Driver interface in package hachi.
package drivers

import (
	_ "github.com/Francesco149/go-hachi/hachi/drivers/termloop"
)

// Package gatearray emulates the CPC Gate Array and related memory control.
package gatearray

import "cpcgo/internal/bus"

const BorderPen = 16

// GateArray handles palette, mode, ROM, and RAM mapping commands.
type GateArray struct {
	memory *bus.Memory

	selectedPen           uint8
	inks                  [17]uint8
	mode                  uint8
	interruptCounterReset bool
}

// New creates a Gate Array connected to the CPC memory map.
func New(memory *bus.Memory) *GateArray {
	return &GateArray{memory: memory}
}

// ReadPort implements bus.IODevice. The classic Gate Array is write-only.
func (g *GateArray) ReadPort(uint16) (uint8, bool) {
	return 0, false
}

// WritePort implements bus.IODevice.
func (g *GateArray) WritePort(port uint16, val uint8) bool {
	if !selectedByPort(port) {
		return false
	}

	switch val & 0xc0 {
	case 0x00:
		g.selectPen(val)
	case 0x40:
		g.setInk(val)
	case 0x80:
		if val&0x20 == 0 {
			g.setRMR(val)
		}
	case 0xc0:
		_ = g.memory.SetRAMConfig(val & 0x07)
	}
	return true
}

// SelectedPen returns the currently selected pen, or BorderPen.
func (g *GateArray) SelectedPen() uint8 {
	return g.selectedPen
}

// Ink returns a hardware colour value for a pen.
func (g *GateArray) Ink(pen uint8) uint8 {
	if pen > BorderPen {
		return 0
	}
	return g.inks[pen]
}

// Mode returns the selected screen mode.
func (g *GateArray) Mode() uint8 {
	return g.mode
}

// InterruptCounterReset reports whether the RMR interrupt reset bit was last
// written as set. Timing work will replace this with a real counter.
func (g *GateArray) InterruptCounterReset() bool {
	return g.interruptCounterReset
}

func (g *GateArray) selectPen(val uint8) {
	if val&0x10 != 0 {
		g.selectedPen = BorderPen
		return
	}
	g.selectedPen = val & 0x0f
}

func (g *GateArray) setInk(val uint8) {
	g.inks[g.selectedPen] = val & 0x1f
}

func (g *GateArray) setRMR(val uint8) {
	g.mode = val & 0x03
	g.memory.SetLowerROMEnabled(val&0x04 == 0)
	g.memory.SetUpperROMEnabled(val&0x08 == 0)
	g.interruptCounterReset = val&0x10 != 0
}

func selectedByPort(port uint16) bool {
	return port&0xc000 == 0x4000
}

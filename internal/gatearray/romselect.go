package gatearray

import "cpcgo/internal/bus"

// ROMSelect handles the CPC upper ROM select latch.
type ROMSelect struct {
	memory *bus.Memory
}

// NewROMSelect creates an upper ROM select device.
func NewROMSelect(memory *bus.Memory) *ROMSelect {
	return &ROMSelect{memory: memory}
}

// ReadPort implements bus.IODevice. The ROM select latch is write-only.
func (r *ROMSelect) ReadPort(uint16) (uint8, bool) {
	return 0, false
}

// WritePort implements bus.IODevice.
func (r *ROMSelect) WritePort(port uint16, val uint8) bool {
	if port&0x2000 != 0 {
		return false
	}
	r.memory.SelectUpperROM(val)
	return true
}

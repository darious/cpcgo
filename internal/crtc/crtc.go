// Package crtc emulates a Motorola 6845-compatible CRTC.
package crtc

const RegisterCount = 18

// CRTC stores the currently programmed 6845 registers.
type CRTC struct {
	selected  uint8
	registers [RegisterCount]uint8
}

// New creates a CRTC.
func New() *CRTC {
	return &CRTC{}
}

// ReadPort implements bus.IODevice.
func (c *CRTC) ReadPort(port uint16) (uint8, bool) {
	if !selectedByPort(port) {
		return 0, false
	}

	switch registerSelect(port) {
	case 2:
		return 0, true
	case 3:
		if c.selected < RegisterCount {
			return c.registers[c.selected], true
		}
		return 0xff, true
	default:
		return 0, false
	}
}

// WritePort implements bus.IODevice.
func (c *CRTC) WritePort(port uint16, val uint8) bool {
	if !selectedByPort(port) {
		return false
	}

	switch registerSelect(port) {
	case 0:
		c.selected = val & 0x1f
	case 1:
		if c.selected < RegisterCount {
			c.registers[c.selected] = val
		}
	default:
		return false
	}
	return true
}

// Selected returns the selected CRTC register.
func (c *CRTC) Selected() uint8 {
	return c.selected
}

// Register returns a CRTC register value.
func (c *CRTC) Register(index uint8) uint8 {
	if index >= RegisterCount {
		return 0
	}
	return c.registers[index]
}

func selectedByPort(port uint16) bool {
	return port&0x4000 == 0
}

func registerSelect(port uint16) uint8 {
	return uint8((port >> 8) & 0x03)
}

// Package crtc emulates the 6845-family CRT controller used in the CPC.
//
// The CRTC is clocked once per character (1 MHz on the CPC). Tick advances
// the counters by one character; after each Tick the outputs (memory address
// MA, raster address RA, display enable, HSYNC and VSYNC) describe the
// character that the Gate Array fetches during the following microsecond.
//
// Counter model (MC6845 datasheet terms):
//   - HCC counts characters 0..R0; display is enabled until HCC reaches R1;
//     HSYNC starts when HCC reaches R2 and lasts R3 bits 0-3 characters.
//   - VLC counts scanlines 0..R9 within a character row; VCC counts rows
//     0..R4, followed by R5 vertical-adjust scanlines.
//   - Vertical display ends when VCC reaches R6; VSYNC starts on the first
//     scanline of row R7.
//   - MA starts each frame at R12/R13 and advances by R1 per character row.
//
// Types follow the usual CPC numbering: 0 = HD6845S/UM6845, 1 = UM6845R,
// 2 = MC6845, 3 = AMS40489 (CPC Plus ASIC), 4 = AMS40226 (pre-ASIC).
package crtc

const RegisterCount = 18

// Write masks for R0-R17 (unused bits read back as zero).
var registerMasks = [RegisterCount]uint8{
	0xff, 0xff, 0xff, 0xff, 0x7f, 0x1f, 0x7f, 0x7f,
	0xf3, 0x1f, 0x7f, 0x1f, 0x3f, 0xff, 0x3f, 0xff, 0x3f, 0xff,
}

// CRTC is a 6845 character clock state machine.
type CRTC struct {
	typ       int
	selected  uint8
	registers [RegisterCount]uint8

	hcc, vlc, vcc uint8
	vtac          uint8
	inAdjust      bool
	hsc, vsc      uint8

	hdisp, vdisp bool
	hsync, vsync bool

	ma, rowStart, nextRowStart uint16
}

// New creates a CRTC of the given type (0-4) with power-on register values
// that produce a PAL-like display until the firmware programs it.
func New() *CRTC {
	return NewType(1)
}

// NewType creates a CRTC of a specific type.
func NewType(typ int) *CRTC {
	c := &CRTC{typ: typ}
	c.registers[0] = 63
	c.registers[1] = 40
	c.registers[2] = 46
	c.registers[3] = 0x8e
	c.registers[4] = 38
	c.registers[6] = 25
	c.registers[7] = 30
	c.registers[9] = 7
	c.hdisp, c.vdisp = true, true
	return c
}

// Type returns the CRTC type number.
func (c *CRTC) Type() int { return c.typ }

// SetType changes the CRTC type.
func (c *CRTC) SetType(typ int) { c.typ = typ }

// ReadPort implements bus.IODevice. The CRTC is selected when A14 is low;
// A9..A8 select the function.
func (c *CRTC) ReadPort(port uint16) (uint8, bool) {
	if port&0x4000 != 0 {
		return 0, false
	}
	switch (port >> 8) & 0x03 {
	case 2:
		return c.status(), true
	case 3:
		return c.readRegister(), true
	}
	return 0, false
}

// WritePort implements bus.IODevice.
func (c *CRTC) WritePort(port uint16, val uint8) bool {
	if port&0x4000 != 0 {
		return false
	}
	switch (port >> 8) & 0x03 {
	case 0:
		c.selected = val & 0x1f
	case 1:
		c.writeRegister(val)
	default:
		return false
	}
	return true
}

func (c *CRTC) writeRegister(val uint8) {
	if c.selected >= 16 {
		return // R16/R17 are the read-only light pen registers
	}
	c.registers[c.selected] = val & registerMasks[c.selected]
}

// readRegister implements the type-dependent register read port (&BFxx).
func (c *CRTC) readRegister() uint8 {
	r := c.selected
	switch c.typ {
	case 0:
		if r >= 12 && r < RegisterCount {
			return c.registers[r]
		}
	case 1, 2:
		if r >= 14 && r < RegisterCount {
			return c.registers[r]
		}
	default: // 3, 4: R12-R17 readable, R0-R7 mirror R8-R15 on reads
		if r >= 12 && r < RegisterCount {
			return c.registers[r]
		}
		if r < 8 {
			return c.registers[r+8]
		}
	}
	return 0
}

// status implements the &BExx read port. Only the UM6845R (type 1) has a
// status register; bit 5 is set during vertical blanking.
func (c *CRTC) status() uint8 {
	switch c.typ {
	case 1:
		if !c.vdisp {
			return 0x20
		}
		return 0
	case 3, 4:
		return c.readRegister()
	}
	return 0xff
}

// Selected returns the selected register number.
func (c *CRTC) Selected() uint8 { return c.selected }

// Register returns a register value.
func (c *CRTC) Register(index uint8) uint8 {
	if index >= RegisterCount {
		return 0
	}
	return c.registers[index]
}

// SetRegister sets a register directly (tests and snapshots).
func (c *CRTC) SetRegister(index, val uint8) {
	if index < RegisterCount {
		c.registers[index] = val & registerMasks[index]
	}
}

// MA returns the 14-bit memory address output.
func (c *CRTC) MA() uint16 { return c.ma & 0x3fff }

// RA returns the raster address output (scanline within the character row).
func (c *CRTC) RA() uint8 { return c.vlc & 0x1f }

// DisplayEnabled reports whether the current character is in the display
// area (as opposed to the border).
func (c *CRTC) DisplayEnabled() bool {
	if !c.hdisp || !c.vdisp {
		return false
	}
	if c.typ != 1 && c.registers[8]&0x30 == 0x30 {
		return false // display skew "11" disables the display
	}
	return true
}

// HSync reports whether HSYNC is active.
func (c *CRTC) HSync() bool { return c.hsync }

// VSync reports whether VSYNC is active.
func (c *CRTC) VSync() bool { return c.vsync }

// HCC returns the horizontal character counter.
func (c *CRTC) HCC() uint8 { return c.hcc }

// VCC returns the vertical character row counter.
func (c *CRTC) VCC() uint8 { return c.vcc }

// VLC returns the scanline counter within the row.
func (c *CRTC) VLC() uint8 { return c.vlc }

func (c *CRTC) hsyncWidth() uint8 {
	w := c.registers[3] & 0x0f
	if w == 0 && c.typ >= 2 {
		w = 16
	}
	return w
}

func (c *CRTC) vsyncWidth() uint8 {
	if c.typ == 1 || c.typ == 2 {
		return 16
	}
	w := c.registers[3] >> 4
	if w == 0 {
		w = 16
	}
	return w
}

// Tick advances the CRTC by one character clock.
func (c *CRTC) Tick() {
	if c.hsync {
		c.hsc++
		if c.hsc >= c.hsyncWidth() {
			c.hsync = false
		}
	}

	if c.hcc == c.registers[0] {
		c.hcc = 0
		c.hdisp = true
		c.endOfLine()
	} else {
		c.hcc++
		c.ma++
	}

	if c.hcc == c.registers[1] {
		c.hdisp = false
		if c.vlc == c.registers[9] && !c.inAdjust {
			c.nextRowStart = c.ma
		}
	}
	if c.hcc == c.registers[2] && !c.hsync {
		c.hsc = 0
		c.hsync = c.hsyncWidth() != 0
	}
}

func (c *CRTC) endOfLine() {
	if c.vsync {
		c.vsc++
		if c.vsc >= c.vsyncWidth() {
			c.vsync = false
		}
	}

	switch {
	case c.inAdjust:
		c.vtac++
		if c.vtac >= c.registers[5] {
			c.newFrame()
		}
	case c.vlc == c.registers[9]:
		c.vlc = 0
		c.rowStart = c.nextRowStart
		if c.vcc == c.registers[4] {
			if c.registers[5] != 0 {
				c.inAdjust = true
				c.vtac = 0
				c.vcc = (c.vcc + 1) & 0x7f
			} else {
				c.newFrame()
			}
		} else {
			c.vcc = (c.vcc + 1) & 0x7f
		}
	default:
		c.vlc = (c.vlc + 1) & 0x1f
	}

	if c.vlc == 0 && !c.inAdjust {
		// Type 0 still displays the first scanline of a frame when R6=0.
		if c.vcc == c.registers[6] && !(c.typ == 0 && c.vcc == 0) {
			c.vdisp = false
		}
		if c.vcc == c.registers[7] && !c.vsync {
			c.vsync = true
			c.vsc = 0
		}
	}
	if c.typ == 0 && c.registers[6] == 0 && (c.vcc != 0 || c.vlc != 0 || c.inAdjust) {
		c.vdisp = false
	}
	c.ma = c.rowStart
}

func (c *CRTC) newFrame() {
	c.inAdjust = false
	c.vcc = 0
	c.vlc = 0
	c.vdisp = c.registers[6] != 0 || c.typ == 0
	c.rowStart = uint16(c.registers[12])<<8 | uint16(c.registers[13])
	c.nextRowStart = c.rowStart
}

// Package gatearray emulates the CPC Gate Array: palette, screen mode, ROM
// and RAM mapping, the raster interrupt counter, and pixel generation.
package gatearray

import "cpcgo/internal/bus"

const (
	BorderPen = 16

	// Black is the hardware colour the Gate Array outputs while blanking.
	Black = 20

	// PixelsPerTick is the number of mode 2 pixels produced per microsecond.
	PixelsPerTick = 16
)

// VideoSource is the CRTC state the Gate Array samples every microsecond.
type VideoSource interface {
	MA() uint16
	RA() uint8
	DisplayEnabled() bool
	HSync() bool
	VSync() bool
}

// Display receives the Gate Array's video output.
type Display interface {
	// Pixels receives 16 hardware colour numbers for one microsecond.
	Pixels(px *[PixelsPerTick]uint8)
	// HSync and VSync signal the start of the Gate Array's sync pulses.
	HSync()
	VSync()
}

// GateArray handles palette, mode, ROM, and RAM mapping commands and drives
// the display.
type GateArray struct {
	memory *bus.Memory

	selectedPen uint8
	inks        [17]uint8
	mode        uint8 // mode requested through the MRER
	activeMode  uint8 // mode latched at the start of HSYNC

	// Raster interrupt state: a 6-bit counter of HSYNCs, the interrupt
	// request line, and the HSYNC countdown after a VSYNC starts.
	counter     uint8
	irq         bool
	vsyncDelay  uint8
	prevHSync   bool
	prevVSync   bool
	hsyncTicks  int
	vblankLines int

	pixels [PixelsPerTick]uint8
}

// New creates a Gate Array connected to the CPC memory map.
func New(memory *bus.Memory) *GateArray {
	return &GateArray{memory: memory, mode: 1, activeMode: 1}
}

// ReadPort implements bus.IODevice. The Gate Array is write-only.
func (g *GateArray) ReadPort(uint16) (uint8, bool) {
	return 0, false
}

// WritePort implements bus.IODevice. The Gate Array is selected when A15 is
// low and A14 is high.
func (g *GateArray) WritePort(port uint16, val uint8) bool {
	if port&0xc000 != 0x4000 {
		return false
	}
	switch val & 0xc0 {
	case 0x00:
		if val&0x10 != 0 {
			g.selectedPen = BorderPen
		} else {
			g.selectedPen = val & 0x0f
		}
	case 0x40:
		g.inks[g.selectedPen] = val & 0x1f
	case 0x80:
		g.setMRER(val)
	case 0xc0:
		_ = g.memory.SetRAMConfig(val & 0x07)
	}
	return true
}

func (g *GateArray) setMRER(val uint8) {
	g.mode = val & 0x03
	g.memory.SetLowerROMEnabled(val&0x04 == 0)
	g.memory.SetUpperROMEnabled(val&0x08 == 0)
	if val&0x10 != 0 {
		g.counter = 0
		g.irq = false
	}
}

// SelectedPen returns the currently selected pen, or BorderPen.
func (g *GateArray) SelectedPen() uint8 { return g.selectedPen }

// Ink returns the hardware colour assigned to a pen.
func (g *GateArray) Ink(pen uint8) uint8 {
	if pen > BorderPen {
		return 0
	}
	return g.inks[pen]
}

// Mode returns the screen mode most recently selected by software.
func (g *GateArray) Mode() uint8 { return g.mode }

// IRQ reports whether the Gate Array is requesting an interrupt.
func (g *GateArray) IRQ() bool { return g.irq }

// Counter returns the raster interrupt counter.
func (g *GateArray) Counter() uint8 { return g.counter }

// Acknowledge handles the Z80 interrupt acknowledge: the request is cleared
// and bit 5 of the counter is reset, so the next interrupt comes no sooner
// than 32 lines later.
func (g *GateArray) Acknowledge() {
	g.irq = false
	g.counter &= 0x1f
}

// Clock runs one microsecond: it samples the CRTC outputs for the current
// character, updates the interrupt logic on sync edges, and sends sixteen
// pixels and any sync pulses to the display.
func (g *GateArray) Clock(crtc VideoSource, display Display) {
	hsync, vsync := crtc.HSync(), crtc.VSync()

	if vsync && !g.prevVSync {
		g.vsyncDelay = 2
		g.vblankLines = 26
		display.VSync()
	}
	if hsync && !g.prevHSync {
		g.activeMode = g.mode
		g.hsyncTicks = 0
	}
	if hsync {
		g.hsyncTicks++
		// The monitor HSYNC starts two microseconds into the CRTC HSYNC.
		if g.hsyncTicks == 3 {
			display.HSync()
		}
	}
	if !hsync && g.prevHSync {
		g.endOfHSync()
	}
	g.prevHSync, g.prevVSync = hsync, vsync

	g.render(crtc, hsync)
	display.Pixels(&g.pixels)
}

func (g *GateArray) endOfHSync() {
	if g.vblankLines > 0 {
		g.vblankLines--
	}
	g.counter = (g.counter + 1) & 0x3f
	if g.counter == 52 {
		g.counter = 0
		g.irq = true
	}
	if g.vsyncDelay > 0 {
		g.vsyncDelay--
		if g.vsyncDelay == 0 {
			if g.counter >= 32 {
				g.irq = true
			}
			g.counter = 0
		}
	}
}

func (g *GateArray) render(crtc VideoSource, hsync bool) {
	px := &g.pixels
	if hsync || g.vblankLines > 0 {
		fill(px, 0, PixelsPerTick, Black)
		return
	}
	if !crtc.DisplayEnabled() {
		fill(px, 0, PixelsPerTick, g.inks[BorderPen])
		return
	}
	ma, ra := crtc.MA(), crtc.RA()
	addr := (ma&0x3000)<<2 | uint16(ra&7)<<11 | (ma&0x03ff)<<1
	for i := uint16(0); i < 2; i++ {
		g.renderByte(px[i*8:i*8+8], g.memory.VideoRead(addr+i))
	}
}

func (g *GateArray) renderByte(out []uint8, b uint8) {
	switch g.activeMode {
	case 0:
		left, right := Mode0Pens(b)
		fill8(out, 0, 4, g.inks[left])
		fill8(out, 4, 8, g.inks[right])
	case 1:
		pens := Mode1Pens(b)
		for i, pen := range pens {
			out[i*2] = g.inks[pen]
			out[i*2+1] = g.inks[pen]
		}
	case 2:
		for i := 0; i < 8; i++ {
			out[i] = g.inks[(b>>(7-i))&1]
		}
	case 3:
		left := (b>>7)&1 | (b>>2)&2
		right := (b>>6)&1 | (b>>1)&2
		fill8(out, 0, 4, g.inks[left])
		fill8(out, 4, 8, g.inks[right])
	}
}

// Mode0Pens decodes the two 16-colour pixels packed in a mode 0 byte.
func Mode0Pens(b uint8) (uint8, uint8) {
	left := (b>>7)&1 | (b>>2)&2 | (b>>3)&4 | (b<<2)&8
	right := (b>>6)&1 | (b>>1)&2 | (b>>2)&4 | (b<<3)&8
	return left, right
}

// Mode1Pens decodes the four 4-colour pixels packed in a mode 1 byte.
func Mode1Pens(b uint8) [4]uint8 {
	return [4]uint8{
		(b>>7)&1 | (b>>2)&2,
		(b>>6)&1 | (b>>1)&2,
		(b>>5)&1 | b&2,
		(b>>4)&1 | (b<<1)&2,
	}
}

func fill(px *[PixelsPerTick]uint8, from, to int, c uint8) {
	for i := from; i < to; i++ {
		px[i] = c
	}
}

func fill8(out []uint8, from, to int, c uint8) {
	for i := from; i < to; i++ {
		out[i] = c
	}
}

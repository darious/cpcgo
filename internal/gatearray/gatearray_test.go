package gatearray

import (
	"testing"

	"cpcgo/internal/bus"
	"cpcgo/internal/rom"
)

type fakeCRTC struct {
	ma           uint16
	ra           uint8
	disp, hs, vs bool
}

func (f *fakeCRTC) MA() uint16           { return f.ma }
func (f *fakeCRTC) RA() uint8            { return f.ra }
func (f *fakeCRTC) DisplayEnabled() bool { return f.disp }
func (f *fakeCRTC) HSync() bool          { return f.hs }
func (f *fakeCRTC) VSync() bool          { return f.vs }

type sink struct {
	px     [][PixelsPerTick]uint8
	hsyncs int
}

func (s *sink) Pixels(p *[PixelsPerTick]uint8) { s.px = append(s.px, *p) }
func (s *sink) HSync()                         { s.hsyncs++ }
func (s *sink) VSync()                         {}

func newGA(t *testing.T) *GateArray {
	image := rom.Image{LowerOS: make([]byte, rom.BankSize), Basic: make([]byte, rom.BankSize)}
	m, err := bus.NewMemory(image)
	if err != nil {
		t.Fatal(err)
	}
	return New(m)
}

func TestPixelDecoding(t *testing.T) {
	l, r := Mode0Pens(0b10101010)
	if l != 0b1111 || r != 0 {
		t.Fatalf("mode 0 pens = %d,%d, want 15,0", l, r)
	}
	if got := Mode1Pens(0b10001000); got != [4]uint8{3, 0, 0, 0} {
		t.Fatalf("mode 1 pens = %v", got)
	}
}

// HSYNC pulses: an interrupt every 52 lines; the acknowledge clears bit 5 of
// the counter; MRER bit 4 resets it.
func TestInterruptCounter(t *testing.T) {
	g := newGA(t)
	c := &fakeCRTC{}
	s := &sink{}
	line := func() {
		c.hs = true
		g.Clock(c, s)
		c.hs = false
		g.Clock(c, s)
	}
	for i := 0; i < 51; i++ {
		line()
	}
	if g.IRQ() {
		t.Fatal("interrupt before 52 lines")
	}
	line()
	if !g.IRQ() || g.Counter() != 0 {
		t.Fatalf("after 52 lines: irq=%v counter=%d", g.IRQ(), g.Counter())
	}
	for i := 0; i < 40; i++ {
		line()
	}
	g.Acknowledge()
	if g.IRQ() || g.Counter() != 8 {
		t.Fatalf("after acknowledge: irq=%v counter=%d, want false 8", g.IRQ(), g.Counter())
	}
	g.WritePort(0x7f00, 0x9c) // MRER with interrupt reset
	if g.Counter() != 0 {
		t.Fatalf("counter after MRER reset = %d", g.Counter())
	}
}

// VSYNC: two HSYNCs later the counter is reset, with an interrupt if it had
// reached 32.
func TestVSyncResynchronisation(t *testing.T) {
	g := newGA(t)
	c := &fakeCRTC{}
	s := &sink{}
	line := func() {
		c.hs = true
		g.Clock(c, s)
		c.hs = false
		g.Clock(c, s)
	}
	for i := 0; i < 40; i++ {
		line()
	}
	c.vs = true
	line()
	if g.IRQ() {
		t.Fatal("interrupt one line after VSYNC")
	}
	line()
	if !g.IRQ() || g.Counter() != 0 {
		t.Fatalf("two lines after VSYNC: irq=%v counter=%d", g.IRQ(), g.Counter())
	}
}

// Palette writes act on the colour output stage, which shows data fetched
// 1.5 microseconds earlier.
func TestPaletteWritesLeadTheDataPipeline(t *testing.T) {
	g := newGA(t)
	c := &fakeCRTC{}
	s := &sink{}
	g.WritePort(0x7f00, 0x10) // border
	g.WritePort(0x7f00, 0x44) // blue
	for i := 0; i < 3; i++ {
		g.Clock(c, s)
	}
	g.WritePort(0x7f00, 0x4b) // white
	g.Clock(c, s)
	if got := s.px[len(s.px)-1]; got[0] != 0x0b || got[15] != 0x0b {
		t.Fatalf("border pixels after write = %v, want white", got)
	}
}

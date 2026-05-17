package video

import (
	"testing"

	"cpcgo/internal/bus"
	"cpcgo/internal/crtc"
	"cpcgo/internal/gatearray"
	"cpcgo/internal/rom"
)

func TestRenderMode1UsesRAMBehindROMOverlay(t *testing.T) {
	memory, ga, c := newRendererTestMachine(t)
	ga.WritePort(0x7f00, 0x00)
	ga.WritePort(0x7f00, 0x40|20) // pen 0 black
	ga.WritePort(0x7f00, 0x01)
	ga.WritePort(0x7f00, 0x40|11) // pen 1 white
	memory.Write(0xc000, 0x88)    // Mode 1 first two pixels use pen 3/0-ish pattern.

	img := Render(memory, c, ga)
	if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
		t.Fatalf("bounds = %v, want %dx%d", img.Bounds(), Width, Height)
	}
	if got := img.RGBAAt(0, 0).A; got != 0xff {
		t.Fatalf("alpha = %#02x, want %#02x", got, 0xff)
	}
}

func TestRenderMode2Pixels(t *testing.T) {
	memory, ga, c := newRendererTestMachine(t)
	ga.WritePort(0x7f00, 0x80|0x02) // mode 2
	ga.WritePort(0x7f00, 0x00)
	ga.WritePort(0x7f00, 0x40|20) // pen 0 black
	ga.WritePort(0x7f00, 0x01)
	ga.WritePort(0x7f00, 0x40|11) // pen 1 white
	memory.Write(0xc000, 0x80)

	img := Render(memory, c, ga)
	if got := img.RGBAAt(0, 0); got != hardwarePalette[11] {
		t.Fatalf("pixel 0 = %#v, want white", got)
	}
	if got := img.RGBAAt(1, 0); got != hardwarePalette[20] {
		t.Fatalf("pixel 1 = %#v, want black", got)
	}
}

func TestScreenBaseUsesCRTCRegisters(t *testing.T) {
	c := crtc.New()
	c.WritePort(0xbc00, 12)
	c.WritePort(0xbd00, 0x30)
	c.WritePort(0xbc00, 13)
	c.WritePort(0xbd00, 0x02)

	if got := screenBase(c); got != 0xc004 {
		t.Fatalf("screen base = %#04x, want %#04x", got, 0xc004)
	}
}

func newRendererTestMachine(t *testing.T) (*bus.Memory, *gatearray.GateArray, *crtc.CRTC) {
	t.Helper()
	image := rom.Image{
		LowerOS: make([]uint8, rom.BankSize),
		Basic:   make([]uint8, rom.BankSize),
		AMSDOS:  make([]uint8, rom.BankSize),
	}
	memory, err := bus.NewMemory(image)
	if err != nil {
		t.Fatal(err)
	}
	c := crtc.New()
	c.WritePort(0xbc00, 12)
	c.WritePort(0xbd00, 0x30)
	ga := gatearray.New(memory)
	return memory, ga, c
}

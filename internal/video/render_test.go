package video

import (
	"image/color"
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

func TestMode0Pens(t *testing.T) {
	left, right := mode0Pens(0x36)
	if left != 12 || right != 6 {
		t.Fatalf("mode 0 pens = %d,%d, want 12,6", left, right)
	}

	left, right = mode0Pens(0xff)
	if left != 15 || right != 15 {
		t.Fatalf("mode 0 all bits = %d,%d, want 15,15", left, right)
	}
}

func TestRenderMode0Pixels(t *testing.T) {
	memory, ga, c := newRendererTestMachine(t)
	ga.WritePort(0x7f00, 0x80) // mode 0
	setInk(t, ga, 6, 18)
	setInk(t, ga, 12, 21)
	memory.Write(0xc000, 0x36)

	img := Render(memory, c, ga)
	if got := img.RGBAAt(0, 0); got != HardwareColor(21) {
		t.Fatalf("left mode 0 pixel = %#v, want pen 12 colour", got)
	}
	if got := img.RGBAAt(3, 0); got != HardwareColor(21) {
		t.Fatalf("left mode 0 run end = %#v, want pen 12 colour", got)
	}
	if got := img.RGBAAt(4, 0); got != HardwareColor(18) {
		t.Fatalf("right mode 0 pixel = %#v, want pen 6 colour", got)
	}
}

func TestMode1Pens(t *testing.T) {
	pens := mode1Pens(0xac)
	want := [4]uint8{3, 2, 1, 0}
	if pens != want {
		t.Fatalf("mode 1 pens = %v, want %v", pens, want)
	}
}

func TestRenderMode1Pixels(t *testing.T) {
	memory, ga, c := newRendererTestMachine(t)
	ga.WritePort(0x7f00, 0x80|0x01) // mode 1
	setInk(t, ga, 0, 20)
	setInk(t, ga, 1, 11)
	setInk(t, ga, 2, 4)
	setInk(t, ga, 3, 18)
	memory.Write(0xc000, 0xac)

	img := Render(memory, c, ga)
	tests := []struct {
		x    int
		ink  uint8
		name string
	}{
		{x: 0, ink: 18, name: "pixel 0"},
		{x: 2, ink: 4, name: "pixel 1"},
		{x: 4, ink: 11, name: "pixel 2"},
		{x: 6, ink: 20, name: "pixel 3"},
	}
	for _, test := range tests {
		if got := img.RGBAAt(test.x, 0); got != HardwareColor(test.ink) {
			t.Fatalf("%s = %#v, want hardware colour %d", test.name, got, test.ink)
		}
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
	if got := img.RGBAAt(0, 0); got != HardwareColor(11) {
		t.Fatalf("pixel 0 = %#v, want white", got)
	}
	if got := img.RGBAAt(1, 0); got != HardwareColor(20) {
		t.Fatalf("pixel 1 = %#v, want black", got)
	}
}

func TestHardwareColorPalette(t *testing.T) {
	tests := map[uint8]color.RGBA{
		0:  {R: 0x80, G: 0x80, B: 0x80, A: 0xff},
		4:  {R: 0x00, G: 0x00, B: 0x80, A: 0xff},
		10: {R: 0xff, G: 0xff, B: 0x00, A: 0xff},
		11: {R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		20: {R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		28: {R: 0x80, G: 0x00, B: 0x00, A: 0xff},
	}

	for hw, want := range tests {
		if got := HardwareColor(hw); got != want {
			t.Fatalf("hardware colour %d = %#v, want %#v", hw, got, want)
		}
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

func setInk(t *testing.T, ga *gatearray.GateArray, pen uint8, ink uint8) {
	t.Helper()
	ga.WritePort(0x7f00, pen)
	ga.WritePort(0x7f00, 0x40|ink)
}

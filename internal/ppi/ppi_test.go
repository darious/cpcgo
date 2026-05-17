package ppi

import "testing"

import "cpcgo/internal/psg"

func TestPPIReadWritePorts(t *testing.T) {
	p := New()

	if !p.WritePort(0xf400, 0x12) {
		t.Fatal("PPI did not match port A")
	}
	p.WritePort(0xf600, 0x34)

	if got, ok := p.ReadPort(0xf400); !ok || got != 0x12 {
		t.Fatalf("port A read = %#02x %v, want %#02x true", got, ok, 0x12)
	}
	if got := p.PortC(); got != 0x34 {
		t.Fatalf("port C = %#02x, want %#02x", got, 0x34)
	}
}

func TestPPIControlAndDecode(t *testing.T) {
	p := New()
	if p.WritePort(0xff00, 0x00) {
		t.Fatal("PPI matched port with bit 11 set")
	}

	p.WritePort(0xf700, 0x82)
	if got := p.Control(); got != 0x82 {
		t.Fatalf("control = %#02x, want %#02x", got, 0x82)
	}

	p.WritePort(0xf700, 0x0f) // Set bit 7 of port C.
	if got := p.PortC(); got != 0x80 {
		t.Fatalf("port C after bit set = %#02x, want %#02x", got, 0x80)
	}
	p.WritePort(0xf700, 0x0e) // Reset bit 7 of port C.
	if got := p.PortC(); got != 0x00 {
		t.Fatalf("port C after bit reset = %#02x, want 0", got)
	}
}

func TestPPIPortBDefault(t *testing.T) {
	p := New()
	got, ok := p.ReadPort(0xf500)
	if !ok {
		t.Fatal("PPI did not match port B")
	}
	if got != 0xfe {
		t.Fatalf("port B default = %#02x, want %#02x", got, 0xfe)
	}
}

func TestPPIDrivesPSGThroughPortAAndC(t *testing.T) {
	sound := psg.New()
	p := New(sound)

	p.WritePort(0xf400, 7)
	p.WritePort(0xf600, 0xc0)
	if got := sound.Selected(); got != 7 {
		t.Fatalf("selected PSG register = %d, want 7", got)
	}

	p.WritePort(0xf400, 0x3f)
	p.WritePort(0xf600, 0x80)
	if got := sound.Register(7); got != 0x3f {
		t.Fatalf("PSG register 7 = %#02x, want %#02x", got, 0x3f)
	}

	p.WritePort(0xf400, 0x00)
	p.WritePort(0xf600, 0x40)
	if got := p.PortA(); got != 0x3f {
		t.Fatalf("PPI port A after PSG read = %#02x, want %#02x", got, 0x3f)
	}
}

func TestPPIKeyboardLine(t *testing.T) {
	p := New()
	p.WritePort(0xf600, 0x0b)
	if got := p.KeyboardLine(); got != 0x0b {
		t.Fatalf("keyboard line = %d, want 11", got)
	}
}

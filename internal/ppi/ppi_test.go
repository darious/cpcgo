package ppi

import "testing"

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

package psg

import "testing"

func TestPSGSelectReadWrite(t *testing.T) {
	p := New()

	p.Select(0x21)
	if got := p.Selected(); got != 1 {
		t.Fatalf("selected = %d, want 1", got)
	}

	p.Write(0x55)
	if got := p.Read(); got != 0x55 {
		t.Fatalf("read = %#02x, want %#02x", got, 0x55)
	}
	if got := p.Register(1); got != 0x55 {
		t.Fatalf("register 1 = %#02x, want %#02x", got, 0x55)
	}
}

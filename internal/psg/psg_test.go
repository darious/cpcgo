package psg

import "testing"

func TestPSGSelectReadWrite(t *testing.T) {
	p := New()

	// Addresses 16-255 select no register.
	p.Select(0x21)
	p.Write(0x55)
	if got := p.Read(); got != 0xff {
		t.Fatalf("read of unselected chip = %#02x, want 0xff", got)
	}

	p.Select(1)
	p.Write(0x55)
	if got := p.Read(); got != 0x05 {
		t.Fatalf("read = %#02x, want 0x05 (R1 is 4 bits)", got)
	}
	p.Select(0)
	p.Write(0x55)
	if got := p.Register(0); got != 0x55 {
		t.Fatalf("register 0 = %#02x, want %#02x", got, 0x55)
	}
}

func TestPSGSetRegister(t *testing.T) {
	p := New()
	p.SetRegister(14, 0x7f)
	if got := p.Register(14); got != 0x7f {
		t.Fatalf("register 14 = %#02x, want %#02x", got, 0x7f)
	}
}

func TestPSGIOPortsDefaultHigh(t *testing.T) {
	p := New()
	if got := p.Register(14); got != 0xff {
		t.Fatalf("register 14 = %#02x, want %#02x", got, 0xff)
	}
	if got := p.Register(15); got != 0xff {
		t.Fatalf("register 15 = %#02x, want %#02x", got, 0xff)
	}
}

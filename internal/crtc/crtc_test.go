package crtc

import "testing"

func TestCRTCRegisterWrites(t *testing.T) {
	c := New()

	if !c.WritePort(0xbc00, 12) {
		t.Fatal("CRTC did not match register select port")
	}
	if !c.WritePort(0xbd00, 0x30) {
		t.Fatal("CRTC did not match register data port")
	}
	if got := c.Selected(); got != 12 {
		t.Fatalf("selected = %d, want 12", got)
	}
	if got := c.Register(12); got != 0x30 {
		t.Fatalf("register 12 = %#02x, want %#02x", got, 0x30)
	}
}

func TestCRTCDecodeAndRead(t *testing.T) {
	c := New()
	if c.WritePort(0xfc00, 1) {
		t.Fatal("CRTC matched port with bit 14 set")
	}

	c.WritePort(0xbc00, 1)
	c.WritePort(0xbd00, 0x20)
	if got, ok := c.ReadPort(0xbf00); !ok || got != 0x20 {
		t.Fatalf("CRTC data read = %#02x %v, want %#02x true", got, ok, 0x20)
	}
	if got, ok := c.ReadPort(0xbe00); !ok || got != 0x00 {
		t.Fatalf("CRTC status read = %#02x %v, want 0 true", got, ok)
	}
}

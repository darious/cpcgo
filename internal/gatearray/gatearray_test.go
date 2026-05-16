package gatearray

import (
	"testing"

	"cpcgo/internal/bus"
	"cpcgo/internal/rom"
)

func TestGateArrayPenInkAndRMR(t *testing.T) {
	memory := testMemory(t)
	gateArray := New(memory)

	if !gateArray.WritePort(0x7f00, 0x02) {
		t.Fatal("Gate Array did not match canonical port")
	}
	gateArray.WritePort(0x7f00, 0x40|0x0b)
	if got := gateArray.Ink(2); got != 0x0b {
		t.Fatalf("pen 2 ink = %#02x, want %#02x", got, 0x0b)
	}

	gateArray.WritePort(0x7f00, 0x10)
	gateArray.WritePort(0x7f00, 0x40|0x1f)
	if got := gateArray.Ink(BorderPen); got != 0x1f {
		t.Fatalf("border ink = %#02x, want %#02x", got, 0x1f)
	}

	gateArray.WritePort(0x7f00, 0x80|0x10|0x08|0x04|0x02)
	if got := gateArray.Mode(); got != 2 {
		t.Fatalf("mode = %d, want 2", got)
	}
	if memory.LowerROMEnabled() {
		t.Fatal("lower ROM still enabled after RMR disable bit")
	}
	if memory.UpperROMEnabled() {
		t.Fatal("upper ROM still enabled after RMR disable bit")
	}
	if !gateArray.InterruptCounterReset() {
		t.Fatal("interrupt counter reset bit was not recorded")
	}
}

func TestGateArrayMMRAndDecode(t *testing.T) {
	memory := testMemory(t)
	gateArray := New(memory)

	if gateArray.WritePort(0xbf00, 0xc1) {
		t.Fatal("Gate Array matched non-GA port")
	}
	gateArray.WritePort(0x4000, 0xc1)
	if got := memory.RAMConfig(); got != 1 {
		t.Fatalf("RAM config = %d, want 1", got)
	}
}

func TestROMSelect(t *testing.T) {
	memory := testMemory(t)
	romSelect := NewROMSelect(memory)

	if !romSelect.WritePort(0xdf00, bus.UpperROMAMSDOS) {
		t.Fatal("ROM select did not match canonical port")
	}
	if got := memory.SelectedUpperROM(); got != bus.UpperROMAMSDOS {
		t.Fatalf("selected upper ROM = %d, want %d", got, bus.UpperROMAMSDOS)
	}
	if romSelect.WritePort(0xff00, 3) {
		t.Fatal("ROM select matched port with bit 13 set")
	}
}

func testMemory(t *testing.T) *bus.Memory {
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
	return memory
}

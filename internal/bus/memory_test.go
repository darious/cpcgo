package bus

import (
	"testing"

	"cpcgo/internal/rom"
)

func TestMemoryROMOverlaysAndWriteThrough(t *testing.T) {
	m := newTestMemory(t)

	m.Write(0x0000, 0x11)
	m.Write(0xc000, 0x22)

	if got := m.Read(0x0000); got != 0xa0 {
		t.Fatalf("lower ROM read = %#02x, want %#02x", got, 0xa0)
	}
	if got := m.Read(0xc000); got != 0xb0 {
		t.Fatalf("upper BASIC ROM read = %#02x, want %#02x", got, 0xb0)
	}

	m.SetLowerROMEnabled(false)
	m.SetUpperROMEnabled(false)

	if got := m.Read(0x0000); got != 0x11 {
		t.Fatalf("lower RAM write-through read = %#02x, want %#02x", got, 0x11)
	}
	if got := m.Read(0xc000); got != 0x22 {
		t.Fatalf("upper RAM write-through read = %#02x, want %#02x", got, 0x22)
	}
}

func TestMemoryUpperROMSelection(t *testing.T) {
	m := newTestMemory(t)

	m.SelectUpperROM(UpperROMAMSDOS)
	if got := m.Read(0xc000); got != 0xd0 {
		t.Fatalf("AMSDOS ROM read = %#02x, want %#02x", got, 0xd0)
	}

	m.SelectUpperROM(3)
	m.Write(0xc000, 0x44)
	if got := m.Read(0xc000); got != 0x44 {
		t.Fatalf("missing upper ROM read = %#02x, want underlying RAM %#02x", got, 0x44)
	}
}

func TestMemoryRAMConfigurations(t *testing.T) {
	m := newTestMemory(t)
	m.SetLowerROMEnabled(false)
	m.SetUpperROMEnabled(false)

	for bank := uint8(0); bank < RAMBankCount; bank++ {
		m.RAMWrite(bank, 0, 0x80+bank)
	}

	for config, banks := range ramConfigurations {
		if err := m.SetRAMConfig(uint8(config)); err != nil {
			t.Fatal(err)
		}
		addrs := []uint16{0x0000, 0x4000, 0x8000, 0xc000}
		for slot, addr := range addrs {
			want := 0x80 + banks[slot]
			if got := m.Read(addr); got != want {
				t.Fatalf("config %d slot %d read = %#02x, want %#02x", config, slot, got, want)
			}
			if got := m.RAMBankForSlot(uint8(slot)); got != banks[slot] {
				t.Fatalf("config %d slot %d bank = %d, want %d", config, slot, got, banks[slot])
			}
		}
	}
}

func TestMemoryRejectsInvalidROMsAndRAMConfig(t *testing.T) {
	_, err := NewMemory(rom.Image{LowerOS: make([]byte, rom.BankSize-1), Basic: make([]byte, rom.BankSize)})
	if err == nil {
		t.Fatal("NewMemory accepted invalid lower ROM size")
	}

	m := newTestMemory(t)
	if err := m.SetRAMConfig(8); err == nil {
		t.Fatal("SetRAMConfig accepted config 8")
	}
}

func newTestMemory(t *testing.T) *Memory {
	t.Helper()

	lower := make([]uint8, rom.BankSize)
	basic := make([]uint8, rom.BankSize)
	amsdos := make([]uint8, rom.BankSize)
	lower[0] = 0xa0
	basic[0] = 0xb0
	amsdos[0] = 0xd0

	m, err := NewMemory(rom.Image{LowerOS: lower, Basic: basic, AMSDOS: amsdos})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

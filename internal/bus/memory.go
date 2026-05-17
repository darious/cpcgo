// Package bus provides CPC memory and I/O dispatch.
package bus

import (
	"fmt"

	"cpcgo/internal/rom"
)

const (
	RAMBankSize  = rom.BankSize
	RAMBankCount = 8
	SlotCount    = 4

	UpperROMBasic  uint8 = 0
	UpperROMAMSDOS uint8 = 7
)

var ramConfigurations = [8][SlotCount]uint8{
	{0, 1, 2, 3},
	{0, 1, 2, 7},
	{4, 5, 6, 7},
	{0, 3, 2, 7},
	{0, 4, 2, 3},
	{0, 5, 2, 3},
	{0, 6, 2, 3},
	{0, 7, 2, 3},
}

// Memory models CPC6128 RAM banking and ROM overlays.
type Memory struct {
	ram [RAMBankCount][RAMBankSize]uint8

	lowerROM  []uint8
	upperROMs map[uint8][]uint8

	ramConfig        uint8
	lowerROMEnabled  bool
	upperROMEnabled  bool
	selectedUpperROM uint8
}

// NewMemory creates CPC6128 memory from validated ROM image banks.
func NewMemory(image rom.Image) (*Memory, error) {
	if len(image.LowerOS) != rom.BankSize {
		return nil, fmt.Errorf("lower ROM size = %d, want %d", len(image.LowerOS), rom.BankSize)
	}
	if len(image.Basic) != rom.BankSize {
		return nil, fmt.Errorf("BASIC ROM size = %d, want %d", len(image.Basic), rom.BankSize)
	}
	if len(image.AMSDOS) != 0 && len(image.AMSDOS) != rom.BankSize {
		return nil, fmt.Errorf("AMSDOS ROM size = %d, want %d", len(image.AMSDOS), rom.BankSize)
	}

	m := &Memory{
		lowerROM:         cloneBank(image.LowerOS),
		upperROMs:        make(map[uint8][]uint8),
		lowerROMEnabled:  true,
		upperROMEnabled:  true,
		selectedUpperROM: UpperROMBasic,
	}
	m.upperROMs[UpperROMBasic] = cloneBank(image.Basic)
	if len(image.AMSDOS) != 0 {
		m.upperROMs[UpperROMAMSDOS] = cloneBank(image.AMSDOS)
	}

	return m, nil
}

// Fetch reads memory during an M1 opcode fetch cycle.
func (m *Memory) Fetch(addr uint16) uint8 {
	return m.Read(addr)
}

// Read reads from the CPU-visible 64K address space.
func (m *Memory) Read(addr uint16) uint8 {
	slot, offset := slotAndOffset(addr)
	if slot == 0 && m.lowerROMEnabled {
		return m.lowerROM[offset]
	}
	if slot == 3 && m.upperROMEnabled {
		if bank, ok := m.upperROMs[m.selectedUpperROM]; ok {
			return bank[offset]
		}
	}

	return m.ram[m.RAMBankForSlot(slot)][offset]
}

// ReadRAM reads the underlying CPU-visible RAM without ROM overlays.
func (m *Memory) ReadRAM(addr uint16) uint8 {
	slot, offset := slotAndOffset(addr)
	return m.ram[m.RAMBankForSlot(slot)][offset]
}

// Write writes to underlying RAM. ROM overlays do not block writes.
func (m *Memory) Write(addr uint16, val uint8) {
	slot, offset := slotAndOffset(addr)
	m.ram[m.RAMBankForSlot(slot)][offset] = val
}

// SetRAMConfig selects one of the CPC6128 RAM configurations 0-7.
func (m *Memory) SetRAMConfig(config uint8) error {
	if config >= uint8(len(ramConfigurations)) {
		return fmt.Errorf("RAM config %d out of range", config)
	}
	m.ramConfig = config
	return nil
}

// RAMConfig returns the active RAM configuration number.
func (m *Memory) RAMConfig() uint8 {
	return m.ramConfig
}

// RAMBankForSlot returns the physical RAM bank mapped into a 16K CPU slot.
func (m *Memory) RAMBankForSlot(slot uint8) uint8 {
	if slot >= SlotCount {
		return 0
	}
	return ramConfigurations[m.ramConfig][slot]
}

// SetLowerROMEnabled controls the lower ROM overlay.
func (m *Memory) SetLowerROMEnabled(enabled bool) {
	m.lowerROMEnabled = enabled
}

// LowerROMEnabled reports whether the lower ROM overlay is active.
func (m *Memory) LowerROMEnabled() bool {
	return m.lowerROMEnabled
}

// SetUpperROMEnabled controls the upper ROM overlay.
func (m *Memory) SetUpperROMEnabled(enabled bool) {
	m.upperROMEnabled = enabled
}

// UpperROMEnabled reports whether the upper ROM overlay is active.
func (m *Memory) UpperROMEnabled() bool {
	return m.upperROMEnabled
}

// SelectUpperROM selects the active upper ROM bank.
func (m *Memory) SelectUpperROM(bank uint8) {
	m.selectedUpperROM = bank
}

// SelectedUpperROM returns the selected upper ROM bank.
func (m *Memory) SelectedUpperROM() uint8 {
	return m.selectedUpperROM
}

// HasUpperROM reports whether an upper ROM bank is installed.
func (m *Memory) HasUpperROM(bank uint8) bool {
	_, ok := m.upperROMs[bank]
	return ok
}

// RAMRead reads a byte from physical RAM for tests and debugger tools.
func (m *Memory) RAMRead(bank uint8, offset uint16) uint8 {
	return m.ram[bank][offset%RAMBankSize]
}

// RAMWrite writes a byte to physical RAM for tests and debugger tools.
func (m *Memory) RAMWrite(bank uint8, offset uint16, val uint8) {
	m.ram[bank][offset%RAMBankSize] = val
}

func slotAndOffset(addr uint16) (uint8, uint16) {
	return uint8(addr >> 14), addr & (RAMBankSize - 1)
}

func cloneBank(data []uint8) []uint8 {
	out := make([]uint8, len(data))
	copy(out, data)
	return out
}

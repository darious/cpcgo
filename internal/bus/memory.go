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

// ramConfigurations lists the physical bank in each 16K slot for the eight
// CPC6128 RAM configurations selected through the Gate Array MMR.
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

// Memory models CPC RAM banking and ROM overlays.
type Memory struct {
	ram   [RAMBankCount][RAMBankSize]uint8
	banks int

	lowerROM  []uint8
	upperROMs map[uint8][]uint8

	ramConfig        uint8
	lowerROMEnabled  bool
	upperROMEnabled  bool
	selectedUpperROM uint8

	// readSlots and writeSlots cache the current mapping for each 16K slot.
	readSlots  [SlotCount][]uint8
	writeSlots [SlotCount][]uint8
}

// NewMemory creates CPC memory with 128K of RAM (CPC6128) from validated ROM
// image banks.
func NewMemory(image rom.Image) (*Memory, error) {
	return NewMemoryWithBanks(image, RAMBankCount)
}

// NewMemoryWithBanks creates CPC memory with 4 (64K) or 8 (128K) RAM banks.
func NewMemoryWithBanks(image rom.Image, banks int) (*Memory, error) {
	if banks != 4 && banks != 8 {
		return nil, fmt.Errorf("RAM bank count = %d, want 4 or 8", banks)
	}
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
		banks:           banks,
		lowerROM:        cloneBank(image.LowerOS),
		upperROMs:       make(map[uint8][]uint8),
		lowerROMEnabled: true,
		upperROMEnabled: true,
	}
	m.upperROMs[UpperROMBasic] = cloneBank(image.Basic)
	if len(image.AMSDOS) != 0 {
		m.upperROMs[UpperROMAMSDOS] = cloneBank(image.AMSDOS)
	}
	m.remap()
	return m, nil
}

// SetLowerROM replaces the lower (OS) ROM contents, for direct-boot test ROMs.
func (m *Memory) SetLowerROM(data []uint8) {
	m.lowerROM = make([]uint8, RAMBankSize)
	copy(m.lowerROM, data)
	m.remap()
}

// LowerROM returns the lower ROM contents.
func (m *Memory) LowerROM() []uint8 { return m.lowerROM }

// SetUpperROM installs or replaces an upper ROM bank.
func (m *Memory) SetUpperROM(bank uint8, data []uint8) {
	b := make([]uint8, RAMBankSize)
	copy(b, data)
	m.upperROMs[bank] = b
	m.remap()
}

func (m *Memory) remap() {
	for slot := uint8(0); slot < SlotCount; slot++ {
		bank := m.RAMBankForSlot(slot)
		m.writeSlots[slot] = m.ram[bank][:]
		m.readSlots[slot] = m.ram[bank][:]
	}
	if m.lowerROMEnabled {
		m.readSlots[0] = m.lowerROM
	}
	if m.upperROMEnabled {
		m.readSlots[3] = m.upperROM()
	}
}

// upperROM returns the selected upper ROM. ROM numbers without an installed
// expansion ROM select BASIC, as on the CPC where BASIC answers any number
// that no expansion claims.
func (m *Memory) upperROM() []uint8 {
	if bank, ok := m.upperROMs[m.selectedUpperROM]; ok {
		return bank
	}
	return m.upperROMs[UpperROMBasic]
}

// Fetch reads memory during an M1 opcode fetch cycle.
func (m *Memory) Fetch(addr uint16) uint8 {
	return m.readSlots[addr>>14][addr&(RAMBankSize-1)]
}

// Read reads from the CPU-visible 64K address space.
func (m *Memory) Read(addr uint16) uint8 {
	return m.readSlots[addr>>14][addr&(RAMBankSize-1)]
}

// ReadRAM reads the underlying CPU-visible RAM without ROM overlays.
func (m *Memory) ReadRAM(addr uint16) uint8 {
	return m.writeSlots[addr>>14][addr&(RAMBankSize-1)]
}

// Write writes to underlying RAM. ROM overlays do not block writes.
func (m *Memory) Write(addr uint16, val uint8) {
	m.writeSlots[addr>>14][addr&(RAMBankSize-1)] = val
}

// VideoRead reads the byte the Gate Array fetches for the display. The video
// address always refers to the base 64K (banks 0-3).
func (m *Memory) VideoRead(addr uint16) uint8 {
	return m.ram[addr>>14][addr&(RAMBankSize-1)]
}

// SetRAMConfig selects one of the CPC6128 RAM configurations 0-7. 64K
// machines ignore it.
func (m *Memory) SetRAMConfig(config uint8) error {
	if config >= uint8(len(ramConfigurations)) {
		return fmt.Errorf("RAM config %d out of range", config)
	}
	if m.banks == RAMBankCount {
		m.ramConfig = config
		m.remap()
	}
	return nil
}

// RAMConfig returns the active RAM configuration number.
func (m *Memory) RAMConfig() uint8 {
	return m.ramConfig
}

// RAMBanks returns the number of 16K RAM banks fitted.
func (m *Memory) RAMBanks() int {
	return m.banks
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
	if m.lowerROMEnabled != enabled {
		m.lowerROMEnabled = enabled
		m.remap()
	}
}

// LowerROMEnabled reports whether the lower ROM overlay is active.
func (m *Memory) LowerROMEnabled() bool {
	return m.lowerROMEnabled
}

// SetUpperROMEnabled controls the upper ROM overlay.
func (m *Memory) SetUpperROMEnabled(enabled bool) {
	if m.upperROMEnabled != enabled {
		m.upperROMEnabled = enabled
		m.remap()
	}
}

// UpperROMEnabled reports whether the upper ROM overlay is active.
func (m *Memory) UpperROMEnabled() bool {
	return m.upperROMEnabled
}

// SelectUpperROM selects the active upper ROM bank.
func (m *Memory) SelectUpperROM(bank uint8) {
	m.selectedUpperROM = bank
	m.remap()
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

// PhysicalRAM returns a copy of the fitted RAM banks in physical order.
func (m *Memory) PhysicalRAM() []uint8 {
	out := make([]uint8, 0, m.banks*RAMBankSize)
	for bank := 0; bank < m.banks; bank++ {
		out = append(out, m.ram[bank][:]...)
	}
	return out
}

func cloneBank(data []uint8) []uint8 {
	out := make([]uint8, len(data))
	copy(out, data)
	return out
}

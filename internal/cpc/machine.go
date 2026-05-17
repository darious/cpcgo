// Package cpc coordinates the emulated CPC machine.
package cpc

import (
	"fmt"

	"cpcgo/internal/bus"
	"cpcgo/internal/crtc"
	"cpcgo/internal/gatearray"
	"cpcgo/internal/ppi"
	"cpcgo/internal/psg"
	"cpcgo/internal/rom"
	"cpcgo/internal/z80"
)

// Model identifies the CPC model being emulated.
type Model string

const Model6128 Model = "6128"

// Config contains startup configuration for a CPC machine.
type Config struct {
	Model Model
	ROMs  rom.Image
	Disk  string
	Scale int
}

// Machine is the top-level emulator state.
type Machine struct {
	config Config
	memory *bus.Memory
	io     *bus.IO
	bus    *bus.Bus
	cpu    *z80.CPU

	gateArray *gatearray.GateArray
	romSelect *gatearray.ROMSelect
	crtc      *crtc.CRTC
	ppi       *ppi.PPI
	psg       *psg.PSG

	timing       timingState
	interruptSet bool
}

// New constructs a machine from validated configuration.
func New(config Config) (*Machine, error) {
	if config.Model == "" {
		config.Model = Model6128
	}
	if config.Model != Model6128 {
		return nil, fmt.Errorf("unsupported model %q", config.Model)
	}

	memory, err := bus.NewMemory(config.ROMs)
	if err != nil {
		return nil, err
	}
	io := bus.NewIO()
	gateArray := gatearray.New(memory)
	romSelect := gatearray.NewROMSelect(memory)
	crtcDevice := crtc.New()
	psgDevice := psg.New()
	ppiDevice := ppi.New(psgDevice)
	io.Add(gateArray)
	io.Add(romSelect)
	io.Add(crtcDevice)
	io.Add(ppiDevice)

	cpcBus := bus.New(memory, io)

	return &Machine{
		config:    config,
		memory:    memory,
		io:        io,
		bus:       cpcBus,
		cpu:       z80.New(cpcBus),
		gateArray: gateArray,
		romSelect: romSelect,
		crtc:      crtcDevice,
		ppi:       ppiDevice,
		psg:       psgDevice,
	}, nil
}

// Config returns the machine startup configuration.
func (m *Machine) Config() Config {
	return m.config
}

// Memory returns the machine memory map.
func (m *Machine) Memory() *bus.Memory {
	return m.memory
}

// IO returns the machine I/O dispatcher.
func (m *Machine) IO() *bus.IO {
	return m.io
}

// Bus returns the CPU-facing bus.
func (m *Machine) Bus() *bus.Bus {
	return m.bus
}

// CPU returns the Z80 CPU adapter.
func (m *Machine) CPU() *z80.CPU {
	return m.cpu
}

// GateArray returns the machine Gate Array.
func (m *Machine) GateArray() *gatearray.GateArray {
	return m.gateArray
}

// CRTC returns the machine CRTC.
func (m *Machine) CRTC() *crtc.CRTC {
	return m.crtc
}

// PPI returns the machine PPI.
func (m *Machine) PPI() *ppi.PPI {
	return m.ppi
}

// PSG returns the machine PSG.
func (m *Machine) PSG() *psg.PSG {
	return m.psg
}

// Timing returns coarse machine timing counters.
func (m *Machine) Timing() TimingStats {
	return m.timing.stats()
}

// Reset resets CPU state. Memory and devices keep their current state.
func (m *Machine) Reset() {
	m.cpu.Reset()
	m.cpu.INT(false, 0xff)
	m.timing.reset()
	m.interruptSet = false
	m.ppi.SetVSync(false)
}

// Step executes one CPU instruction or interrupt service.
func (m *Machine) Step() int {
	cycles := m.cpu.Step()
	if m.interruptSet {
		m.cpu.INT(false, 0xff)
		m.interruptSet = false
	}

	if m.timing.advance(cycles) {
		m.cpu.INT(true, 0xff)
		m.interruptSet = true
	}
	m.ppi.SetVSync(m.timing.vsync)

	return cycles
}

// RunInstructions executes count CPU steps and returns consumed T-states.
func (m *Machine) RunInstructions(count int) uint64 {
	var cycles uint64
	for i := 0; i < count; i++ {
		cycles += uint64(m.Step())
	}
	return cycles
}

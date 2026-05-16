// Package cpc coordinates the emulated CPC machine.
package cpc

import (
	"fmt"

	"cpcgo/internal/bus"
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
	cpcBus := bus.New(memory, io)

	return &Machine{
		config: config,
		memory: memory,
		io:     io,
		bus:    cpcBus,
		cpu:    z80.New(cpcBus),
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

// Reset resets CPU state. Memory and devices keep their current state.
func (m *Machine) Reset() {
	m.cpu.Reset()
}

// Step executes one CPU instruction or interrupt service.
func (m *Machine) Step() int {
	return m.cpu.Step()
}

// RunInstructions executes count CPU steps and returns consumed T-states.
func (m *Machine) RunInstructions(count int) uint64 {
	var cycles uint64
	for i := 0; i < count; i++ {
		cycles += uint64(m.Step())
	}
	return cycles
}

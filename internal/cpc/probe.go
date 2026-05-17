package cpc

import "cpcgo/internal/bus"

// ProbeOptions configures a bounded headless emulator run.
type ProbeOptions struct {
	Instructions int
}

// ProbeResult summarizes a bounded headless run.
type ProbeResult struct {
	Instructions int
	Cycles       uint64
	Registers    ProbeRegisters
	Halted       bool
	IO           IOStats
	Timing       TimingStats
}

// ProbeRegisters contains the CPU state most useful for early boot probing.
type ProbeRegisters struct {
	AF uint16
	BC uint16
	DE uint16
	HL uint16
	IX uint16
	IY uint16
	SP uint16
	PC uint16
	I  uint8
	R  uint8
	IM uint8
}

// IOStats summarizes I/O activity during a probe.
type IOStats struct {
	Reads           int
	Writes          int
	UnhandledReads  int
	UnhandledWrites int
	ReadPorts       map[uint16]int
	WritePorts      map[uint16]int
	UnhandledPorts  map[uint16]int
}

// Probe runs a bounded number of CPU instructions and collects diagnostics.
func (m *Machine) Probe(options ProbeOptions) ProbeResult {
	if options.Instructions < 0 {
		options.Instructions = 0
	}

	stats := IOStats{
		ReadPorts:      make(map[uint16]int),
		WritePorts:     make(map[uint16]int),
		UnhandledPorts: make(map[uint16]int),
	}
	previousObserver := m.io.SetObserver(func(event bus.IOEvent) {
		switch event.Operation {
		case bus.IORead:
			stats.Reads++
			stats.ReadPorts[event.Port]++
			if !event.Handled {
				stats.UnhandledReads++
				stats.UnhandledPorts[event.Port]++
			}
		case bus.IOWrite:
			stats.Writes++
			stats.WritePorts[event.Port]++
			if !event.Handled {
				stats.UnhandledWrites++
				stats.UnhandledPorts[event.Port]++
			}
		}
	})
	defer m.io.SetObserver(previousObserver)

	var cycles uint64
	for i := 0; i < options.Instructions; i++ {
		cycles += uint64(m.Step())
	}

	regs := m.cpu.Registers()
	return ProbeResult{
		Instructions: options.Instructions,
		Cycles:       cycles,
		Registers: ProbeRegisters{
			AF: regs.AF,
			BC: regs.BC,
			DE: regs.DE,
			HL: regs.HL,
			IX: regs.IX,
			IY: regs.IY,
			SP: regs.SP,
			PC: regs.PC,
			I:  regs.I,
			R:  regs.R,
			IM: regs.IM,
		},
		Halted: m.cpu.Halted(),
		IO:     stats,
		Timing: m.Timing(),
	}
}

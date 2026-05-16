// Package z80 adapts the selected Z80 CPU core for cpcgo.
package z80

import chipz80 "github.com/user-none/go-chip-z80"

// Bus provides memory and I/O access for the Z80 CPU.
//
// Fetch is used for M1 opcode fetch cycles. CPC hardware can distinguish M1
// from ordinary memory reads, so the adapter keeps that signal visible.
type Bus interface {
	Fetch(addr uint16) uint8
	Read(addr uint16) uint8
	Write(addr uint16, val uint8)
	In(port uint16) uint8
	Out(port uint16, val uint8)
}

// Registers is a snapshot of the Z80 programmer-visible state.
type Registers = chipz80.Registers

// CPU wraps the selected Z80 implementation.
type CPU struct {
	core *chipz80.CPU
}

// New creates a reset Z80 CPU wired to bus.
func New(bus Bus) *CPU {
	return &CPU{core: chipz80.New(bus)}
}

// Reset reinitializes the CPU to power-on state.
func (c *CPU) Reset() {
	c.core.Reset()
}

// Step executes one instruction or interrupt service and returns consumed
// T-states.
func (c *CPU) Step() int {
	return c.core.Step()
}

// StepCycles executes within a cycle budget and carries deficit forward when
// an instruction exceeds the requested budget.
func (c *CPU) StepCycles(budget int) int {
	return c.core.StepCycles(budget)
}

// AddCycles advances CPU time without executing instructions.
func (c *CPU) AddCycles(cycles uint64) {
	c.core.AddCycles(cycles)
}

// Cycles returns total T-states since reset.
func (c *CPU) Cycles() uint64 {
	return c.core.Cycles()
}

// Deficit returns the cycle debt maintained by StepCycles.
func (c *CPU) Deficit() int {
	return c.core.Deficit()
}

// Halted reports whether the CPU is in HALT state.
func (c *CPU) Halted() bool {
	return c.core.Halted()
}

// Registers returns a CPU register snapshot.
func (c *CPU) Registers() Registers {
	return c.core.Registers()
}

// SetState replaces the CPU register state.
func (c *CPU) SetState(regs Registers) {
	c.core.SetState(regs)
}

// INT asserts or deasserts the maskable interrupt line.
func (c *CPU) INT(assert bool, data uint8) {
	c.core.INT(assert, data)
}

// NMI latches a non-maskable interrupt for the next Step.
func (c *CPU) NMI() {
	c.core.NMI()
}

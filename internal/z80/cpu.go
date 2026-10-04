// Package z80 implements a Zilog Z80 CPU core for cpcgo.
//
// The core executes one instruction per Step and models the processor at the
// level of machine cycles: every opcode fetch, memory access, I/O access and
// internal delay advances a T-state counter in the order the real chip
// performs them. That ordering matters on the Amstrad CPC, where the Gate
// Array stretches bus cycles with WAIT so that memory and I/O accesses land on
// microsecond boundaries (see WaitStates).
//
// The programmer-visible behaviour includes the undocumented flags (bits 3 and
// 5), the internal MEMPTR register (WZ) that leaks into BIT n,(HL), the Q
// latch that affects SCF/CCF, and the undocumented opcodes (SLL, IXH/IXL/IYH/
// IYL, DDCB result copies, ED aliases).
package z80

// Bus provides memory and I/O access for the CPU. The CPU's T-state counter
// (Cycles) already includes the access when a Write, In or Out callback runs,
// so devices can synchronise to the exact moment of the access.
type Bus interface {
	Read(addr uint16) uint8
	Write(addr uint16, val uint8)
	In(port uint16) uint8
	Out(port uint16, val uint8)
}

// InterruptAcknowledger is an optional Bus extension. IntAck is called during
// the maskable interrupt acknowledge cycle and returns the data bus value.
type InterruptAcknowledger interface {
	IntAck() uint8
}

// Flag bits in F.
const (
	FlagC  uint8 = 0x01
	FlagN  uint8 = 0x02
	FlagPV uint8 = 0x04
	FlagX  uint8 = 0x08
	FlagH  uint8 = 0x10
	FlagY  uint8 = 0x20
	FlagZ  uint8 = 0x40
	FlagS  uint8 = 0x80
)

// Registers is a snapshot of the programmer-visible CPU state.
type Registers struct {
	AF, BC, DE, HL     uint16
	AF2, BC2, DE2, HL2 uint16
	IX, IY, SP, PC     uint16
	WZ                 uint16
	I, R               uint8
	IFF1, IFF2         bool
	IM                 uint8
	Halted             bool
}

// CPU is a Z80 processor.
type CPU struct {
	a, f, b, c, d, e, h, l         uint8
	a2, f2, b2, c2, d2, e2, h2, l2 uint8
	ix, iy, sp, pc, wz             uint16
	i, r                           uint8
	iff1, iff2                     bool
	im                             uint8
	halted                         bool

	// afterEI suppresses interrupt acceptance for one instruction after EI.
	afterEI bool
	// q holds F when the previous instruction modified the flags, else 0.
	q          uint8
	flagsDirty bool

	intLine    bool
	intData    uint8
	nmiPending bool

	// idx selects HL (0), IX (1) or IY (2) for the instruction being decoded.
	idx int

	cycles uint64

	// WaitStates enables Amstrad CPC bus timing: the Gate Array holds WAIT
	// active except on one T-state in four, so each machine cycle is delayed
	// until the T-state at which the Z80 samples WAIT is a free one.
	WaitStates bool

	bus   Bus
	acker InterruptAcknowledger
}

// New creates a CPU wired to bus and resets it.
func New(bus Bus) *CPU {
	c := &CPU{bus: bus}
	c.acker, _ = bus.(InterruptAcknowledger)
	c.Reset()
	return c
}

// Reset puts the CPU into its power-on state. The cycle counter is kept.
func (c *CPU) Reset() {
	c.pc, c.i, c.r = 0, 0, 0
	c.iff1, c.iff2 = false, false
	c.im = 0
	c.halted = false
	c.afterEI = false
	c.sp = 0xffff
	c.a, c.f = 0xff, 0xff
	c.q = 0
	c.nmiPending = false
}

// Cycles returns the T-states executed so far.
func (c *CPU) Cycles() uint64 { return c.cycles }

// AddCycles advances the T-state counter without executing anything.
func (c *CPU) AddCycles(n uint64) { c.cycles += n }

// Halted reports whether the CPU is executing HALT.
func (c *CPU) Halted() bool { return c.halted }

// INT sets the level of the maskable interrupt line and the value the CPU
// reads from the data bus when it acknowledges (used by IM 0 and IM 2) unless
// the bus implements InterruptAcknowledger.
func (c *CPU) INT(active bool, data uint8) {
	c.intLine = active
	c.intData = data
}

// NMI latches a non-maskable interrupt request.
func (c *CPU) NMI() { c.nmiPending = true }

// Registers returns a register snapshot.
func (c *CPU) Registers() Registers {
	return Registers{
		AF: pair(c.a, c.f), BC: pair(c.b, c.c), DE: pair(c.d, c.e), HL: pair(c.h, c.l),
		AF2: pair(c.a2, c.f2), BC2: pair(c.b2, c.c2), DE2: pair(c.d2, c.e2), HL2: pair(c.h2, c.l2),
		IX: c.ix, IY: c.iy, SP: c.sp, PC: c.pc, WZ: c.wz,
		I: c.i, R: c.r, IFF1: c.iff1, IFF2: c.iff2, IM: c.im, Halted: c.halted,
	}
}

// SetRegisters replaces the register state.
func (c *CPU) SetRegisters(r Registers) {
	c.a, c.f = hi(r.AF), lo(r.AF)
	c.b, c.c = hi(r.BC), lo(r.BC)
	c.d, c.e = hi(r.DE), lo(r.DE)
	c.h, c.l = hi(r.HL), lo(r.HL)
	c.a2, c.f2 = hi(r.AF2), lo(r.AF2)
	c.b2, c.c2 = hi(r.BC2), lo(r.BC2)
	c.d2, c.e2 = hi(r.DE2), lo(r.DE2)
	c.h2, c.l2 = hi(r.HL2), lo(r.HL2)
	c.ix, c.iy, c.sp, c.pc, c.wz = r.IX, r.IY, r.SP, r.PC, r.WZ
	c.i, c.r = r.I, r.R
	c.iff1, c.iff2, c.im, c.halted = r.IFF1, r.IFF2, r.IM, r.Halted
}

func pair(h, l uint8) uint16 { return uint16(h)<<8 | uint16(l) }
func hi(v uint16) uint8      { return uint8(v >> 8) }
func lo(v uint16) uint8      { return uint8(v) }

// ---------------------------------------------------------------------------
// Machine cycles

// waitUntilFree delays a machine cycle so that the T-state at which WAIT is
// sampled (offset T-states after the cycle starts) is the free one: absolute
// T-state numbers congruent to 1 modulo 4.
func (c *CPU) waitUntilFree(offset uint64) {
	if c.WaitStates {
		s := c.cycles + offset
		c.cycles += (5 - s%4) % 4
	}
}

// fetchOpcode performs an M1 cycle (4 T-states, WAIT sampled in T2).
func (c *CPU) fetchOpcode() uint8 {
	c.waitUntilFree(1)
	op := c.bus.Read(c.pc)
	c.pc++
	c.r = (c.r & 0x80) | ((c.r + 1) & 0x7f)
	c.cycles += 4
	return op
}

// read performs a memory read cycle (3 T-states, WAIT sampled in T2).
func (c *CPU) read(addr uint16) uint8 {
	c.waitUntilFree(1)
	c.cycles += 3
	return c.bus.Read(addr)
}

// write performs a memory write cycle (3 T-states, WAIT sampled in T2).
func (c *CPU) write(addr uint16, val uint8) {
	c.waitUntilFree(1)
	c.cycles += 3
	c.bus.Write(addr, val)
}

// in performs an I/O read cycle (4 T-states, WAIT sampled in the automatic
// wait state TW).
func (c *CPU) in(port uint16) uint8 {
	c.waitUntilFree(2)
	c.cycles += 4
	return c.bus.In(port)
}

// out performs an I/O write cycle.
func (c *CPU) out(port uint16, val uint8) {
	c.waitUntilFree(2)
	c.cycles += 4
	c.bus.Out(port, val)
}

// internal adds n T-states during which the CPU does not use the bus.
func (c *CPU) internal(n uint64) { c.cycles += n }

func (c *CPU) imm8() uint8 {
	v := c.read(c.pc)
	c.pc++
	return v
}

func (c *CPU) imm16() uint16 {
	l := c.imm8()
	h := c.imm8()
	return pair(h, l)
}

func (c *CPU) push(v uint16) {
	c.sp--
	c.write(c.sp, hi(v))
	c.sp--
	c.write(c.sp, lo(v))
}

func (c *CPU) pop() uint16 {
	l := c.read(c.sp)
	c.sp++
	h := c.read(c.sp)
	c.sp++
	return pair(h, l)
}

func (c *CPU) setF(v uint8) {
	c.f = v
	c.flagsDirty = true
}

// ---------------------------------------------------------------------------
// Step

// Step executes one instruction, or services a pending interrupt, and returns
// the T-states it took.
func (c *CPU) Step() int {
	start := c.cycles
	lastQ := c.q
	c.flagsDirty = false

	switch {
	case c.nmiPending:
		c.nmiPending = false
		c.leaveHalt()
		c.iff1 = false
		// Dummy opcode fetch (5 T-states), then push PC.
		c.waitUntilFree(1)
		c.r = (c.r & 0x80) | ((c.r + 1) & 0x7f)
		c.cycles += 5
		c.push(c.pc)
		c.pc = 0x0066
		c.wz = c.pc
	case c.intLine && c.iff1 && !c.afterEI:
		c.leaveHalt()
		c.iff1, c.iff2 = false, false
		c.acknowledgeInterrupt()
	default:
		c.afterEI = false
		if c.halted {
			// HALT keeps executing NOPs without advancing PC.
			c.fetchOpcode()
			c.pc--
		} else {
			c.idx = 0
			c.execute(c.fetchOpcode(), lastQ)
		}
	}

	if c.flagsDirty {
		c.q = c.f
	} else {
		c.q = 0
	}
	return int(c.cycles - start)
}

func (c *CPU) leaveHalt() {
	if c.halted {
		c.halted = false
		c.pc++
	}
}

// acknowledgeInterrupt runs the interrupt acknowledge cycle (an M1 cycle with
// two automatic wait states and one extra T-state) and the response for the
// current interrupt mode.
func (c *CPU) acknowledgeInterrupt() {
	c.waitUntilFree(3)
	c.r = (c.r & 0x80) | ((c.r + 1) & 0x7f)
	c.cycles += 7
	data := c.intData
	if c.acker != nil {
		data = c.acker.IntAck()
	}
	switch c.im {
	case 2:
		c.push(c.pc)
		vector := pair(c.i, data)
		l := c.read(vector)
		h := c.read(vector + 1)
		c.pc = pair(h, l)
	default:
		// IM 1, and IM 0 with the RST 38h that the CPC's floating bus
		// supplies (other IM 0 opcodes are treated as RST too).
		c.push(c.pc)
		if c.im == 0 && data&0xc7 == 0xc7 {
			c.pc = uint16(data & 0x38)
		} else {
			c.pc = 0x0038
		}
	}
	c.wz = c.pc
}

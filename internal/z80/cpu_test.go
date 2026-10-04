package z80

import (
	"os"
	"strings"
	"testing"
)

type ioWrite struct {
	port uint16
	val  uint8
}

type testBus struct {
	mem  [0x10000]uint8
	in   map[uint16]uint8
	outs []ioWrite
	ack  int
}

func newTestBus() *testBus {
	return &testBus{in: make(map[uint16]uint8)}
}

func (b *testBus) Read(addr uint16) uint8          { return b.mem[addr] }
func (b *testBus) Write(addr uint16, val uint8)    { b.mem[addr] = val }
func (b *testBus) In(port uint16) uint8            { return b.in[port] }
func (b *testBus) Out(port uint16, val uint8)      { b.outs = append(b.outs, ioWrite{port, val}) }
func (b *testBus) load(addr uint16, code ...uint8) { copy(b.mem[addr:], code) }

func TestCPUExecutesThroughBus(t *testing.T) {
	bus := newTestBus()
	bus.load(0,
		0x3e, 0x12, // LD A,0x12
		0x32, 0x00, 0x20, // LD (0x2000),A
		0xd3, 0x7f, // OUT (0x7f),A
		0xdb, 0xfe, // IN A,(0xfe)
		0x32, 0x01, 0x20, // LD (0x2001),A
		0x76, // HALT
	)
	bus.in[0x12fe] = 0x34

	cpu := New(bus)
	for i := 0; i < 6; i++ {
		cpu.Step()
	}
	if !cpu.Halted() {
		t.Fatal("CPU is not halted after HALT")
	}
	if bus.mem[0x2000] != 0x12 || bus.mem[0x2001] != 0x34 {
		t.Fatalf("memory = %#02x %#02x, want 0x12 0x34", bus.mem[0x2000], bus.mem[0x2001])
	}
	if len(bus.outs) != 1 || bus.outs[0] != (ioWrite{0x127f, 0x12}) {
		t.Fatalf("outs = %+v", bus.outs)
	}
	if pc := cpu.Registers().PC; pc != 0x000c {
		t.Fatalf("halted PC = %#04x, want 0x000c", pc)
	}
}

// Standard Z80 T-state counts without CPC wait states.
func TestInstructionTStates(t *testing.T) {
	cases := []struct {
		name string
		code []uint8
		want int
	}{
		{"NOP", []uint8{0x00}, 4},
		{"LD A,n", []uint8{0x3e, 1}, 7},
		{"LD (HL),n", []uint8{0x36, 1}, 10},
		{"INC HL", []uint8{0x23}, 6},
		{"ADD HL,BC", []uint8{0x09}, 11},
		{"PUSH BC", []uint8{0xc5}, 11},
		{"POP BC", []uint8{0xc1}, 10},
		{"EX (SP),HL", []uint8{0xe3}, 19},
		{"CALL nn", []uint8{0xcd, 0, 0x40}, 17},
		{"RET", []uint8{0xc9}, 10},
		{"RST 38", []uint8{0xff}, 11},
		{"JR e", []uint8{0x18, 0}, 12},
		{"OUT (n),A", []uint8{0xd3, 0}, 11},
		{"OUT (C),A", []uint8{0xed, 0x79}, 12},
		{"LDI", []uint8{0xed, 0xa0}, 16},
		{"OUTI", []uint8{0xed, 0xa3}, 16},
		{"LD A,(IX+d)", []uint8{0xdd, 0x7e, 1}, 19},
		{"LD (IX+d),n", []uint8{0xdd, 0x36, 1, 2}, 19},
		{"INC (IX+d)", []uint8{0xdd, 0x34, 1}, 23},
		{"BIT 0,(IX+d)", []uint8{0xdd, 0xcb, 1, 0x46}, 20},
		{"SET 0,(IX+d)", []uint8{0xdd, 0xcb, 1, 0xc6}, 23},
		{"RLD", []uint8{0xed, 0x6f}, 18},
		{"ADC HL,BC", []uint8{0xed, 0x4a}, 15},
		{"LD SP,IX", []uint8{0xdd, 0xf9}, 10},
	}
	for _, tc := range cases {
		bus := newTestBus()
		bus.load(0x1000, tc.code...)
		cpu := New(bus)
		regs := cpu.Registers()
		regs.PC, regs.SP = 0x1000, 0x8000
		cpu.SetRegisters(regs)
		if got := cpu.Step(); got != tc.want {
			t.Errorf("%s: %d T-states, want %d", tc.name, got, tc.want)
		}
	}
}

// With CPC wait states every instruction takes a whole number of
// microseconds (4 T-states); these are the documented CPC timings in NOPs.
func TestCPCWaitStateTimings(t *testing.T) {
	cases := []struct {
		name string
		code []uint8
		nops int
	}{
		{"NOP", []uint8{0x00}, 1},
		{"LD A,n", []uint8{0x3e, 1}, 2},
		{"LD (HL),n", []uint8{0x36, 1}, 3},
		{"INC HL", []uint8{0x23}, 2},
		{"ADD HL,BC", []uint8{0x09}, 3},
		{"PUSH BC", []uint8{0xc5}, 4},
		{"POP BC", []uint8{0xc1}, 3},
		{"EX (SP),HL", []uint8{0xe3}, 6},
		{"CALL nn", []uint8{0xcd, 0, 0x40}, 5},
		{"RET", []uint8{0xc9}, 3},
		{"RST 38", []uint8{0xff}, 4},
		{"JR e", []uint8{0x18, 0}, 3},
		{"DJNZ taken", []uint8{0x06, 2, 0x10, 0}, 2 + 4},
		{"OUT (n),A", []uint8{0xd3, 0}, 3},
		{"IN A,(n)", []uint8{0xdb, 0}, 3},
		{"OUT (C),A", []uint8{0xed, 0x79}, 4},
		{"IN A,(C)", []uint8{0xed, 0x78}, 4},
		{"LDI", []uint8{0xed, 0xa0}, 5},
		{"OUTI", []uint8{0xed, 0xa3}, 5},
		{"LD A,(IX+d)", []uint8{0xdd, 0x7e, 1}, 5},
		{"LD (IX+d),n", []uint8{0xdd, 0x36, 1, 2}, 6},
		{"INC (IX+d)", []uint8{0xdd, 0x34, 1}, 6},
		{"RLD", []uint8{0xed, 0x6f}, 5},
		{"ADC HL,BC", []uint8{0xed, 0x4a}, 4},
		{"RET NZ taken", []uint8{0xc0}, 4},
		{"RET Z not taken", []uint8{0xc8}, 2},
		{"JP nn", []uint8{0xc3, 0, 0x10}, 3},
	}
	for _, tc := range cases {
		bus := newTestBus()
		bus.load(0x1000, tc.code...)
		cpu := New(bus)
		cpu.WaitStates = true
		regs := cpu.Registers()
		regs.PC, regs.SP, regs.AF = 0x1000, 0x8000, 0
		cpu.SetRegisters(regs)
		start := cpu.Cycles()
		steps := 1
		if strings.HasPrefix(tc.name, "DJNZ") {
			steps = 2
		}
		for i := 0; i < steps; i++ {
			cpu.Step()
		}
		// The next opcode fetch completes the alignment of the last cycle.
		cpu.waitUntilFree(1)
		if got := cpu.Cycles() - start; got != uint64(tc.nops*4) {
			t.Errorf("%s: %d T-states, want %d (%d NOPs)", tc.name, got, tc.nops*4, tc.nops)
		}
	}
}

func TestInterruptModes(t *testing.T) {
	bus := newTestBus()
	bus.load(0, 0xed, 0x56, 0xfb, 0x00, 0x00) // IM 1; EI; NOP; NOP
	cpu := New(bus)
	cpu.SetRegisters(Registers{SP: 0x8000})
	cpu.Step() // IM 1
	cpu.INT(true, 0xff)
	cpu.Step() // EI: no interrupt accepted yet
	cpu.Step() // NOP after EI still runs
	if cpu.Registers().PC != 4 {
		t.Fatalf("PC after EI;NOP = %#04x, want 4", cpu.Registers().PC)
	}
	if got := cpu.Step(); got != 13 {
		t.Fatalf("IM 1 response took %d T-states, want 13", got)
	}
	if regs := cpu.Registers(); regs.PC != 0x38 || regs.IFF1 || regs.SP != 0x7ffe {
		t.Fatalf("after IM 1: %+v", regs)
	}

	bus = newTestBus()
	bus.load(0, 0xed, 0x5e, 0xfb, 0x76) // IM 2; EI; HALT
	bus.load(0x40fe, 0x34, 0x12)
	cpu = New(bus)
	cpu.SetRegisters(Registers{SP: 0x8000, I: 0x40})
	cpu.Step()
	cpu.Step()
	cpu.Step() // HALT
	cpu.Step() // halted NOP
	cpu.INT(true, 0xfe)
	if got := cpu.Step(); got != 19 {
		t.Fatalf("IM 2 response took %d T-states, want 19", got)
	}
	if regs := cpu.Registers(); regs.PC != 0x1234 || regs.Halted {
		t.Fatalf("after IM 2: %+v", regs)
	}
	if bus.mem[0x7ffe] != 0x04 || bus.mem[0x7fff] != 0x00 {
		t.Fatalf("pushed return address = %02x%02x, want 0004", bus.mem[0x7fff], bus.mem[0x7ffe])
	}
}

func TestZEXDOC(t *testing.T) { runZEX(t, "testdata/zexdoc.com") }
func TestZEXALL(t *testing.T) { runZEX(t, "testdata/zexall.com") }

// runZEX runs a CP/M instruction exerciser. It takes several minutes, so it
// only runs when CPCGO_ZEX=1.
func runZEX(t *testing.T, path string) {
	if os.Getenv("CPCGO_ZEX") != "1" {
		t.Skip("set CPCGO_ZEX=1 to run the instruction exercisers")
	}
	program, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bus := newTestBus()
	bus.load(0x0100, program...)
	bus.mem[0x0005] = 0xc9 // BDOS entry: RET after the trap below
	cpu := New(bus)
	cpu.SetRegisters(Registers{PC: 0x0100, SP: 0xf000})

	var output strings.Builder
	for {
		regs := cpu.Registers()
		switch regs.PC {
		case 0x0000:
			text := output.String()
			t.Log("\n" + text)
			if strings.Contains(text, "ERROR") {
				t.Fatalf("%s reported errors", path)
			}
			if !strings.Contains(text, "Tests complete") {
				t.Fatalf("%s did not complete", path)
			}
			return
		case 0x0005:
			switch lo(regs.BC) {
			case 2:
				output.WriteByte(lo(regs.DE))
			case 9:
				for addr := regs.DE; bus.mem[addr] != '$'; addr++ {
					output.WriteByte(bus.mem[addr])
				}
			}
		}
		cpu.Step()
	}
}

package z80

import "testing"

func TestCPUExecutesThroughBus(t *testing.T) {
	bus := newTestBus()
	copy(bus.mem[:], []uint8{
		0x3e, 0x12, // LD A,0x12
		0x32, 0x00, 0x20, // LD (0x2000),A
		0xd3, 0x7f, // OUT (0x7f),A
		0xdb, 0xfe, // IN A,(0xfe)
		0x32, 0x01, 0x20, // LD (0x2001),A
		0x76, // HALT
	})
	bus.in[0x12fe] = 0x34

	cpu := New(bus)
	for i := 0; i < 6; i++ {
		if cycles := cpu.Step(); cycles <= 0 {
			t.Fatalf("step %d returned %d cycles", i, cycles)
		}
	}

	if !cpu.Halted() {
		t.Fatal("CPU is not halted after HALT instruction")
	}
	if bus.mem[0x2000] != 0x12 {
		t.Fatalf("memory[0x2000] = %#02x, want %#02x", bus.mem[0x2000], 0x12)
	}
	if bus.mem[0x2001] != 0x34 {
		t.Fatalf("memory[0x2001] = %#02x, want %#02x", bus.mem[0x2001], 0x34)
	}
	if got := len(bus.outs); got != 1 {
		t.Fatalf("out count = %d, want 1", got)
	}
	if bus.outs[0] != (ioWrite{port: 0x127f, val: 0x12}) {
		t.Fatalf("out = %+v, want port 0x127f val 0x12", bus.outs[0])
	}
	if got := uint8(cpu.Registers().AF >> 8); got != 0x34 {
		t.Fatalf("A = %#02x, want %#02x", got, 0x34)
	}
	if bus.fetches == 0 {
		t.Fatal("CPU did not use M1 fetch bus path")
	}
	if cpu.Cycles() == 0 {
		t.Fatal("CPU did not accumulate cycles")
	}
}

func TestStepCyclesTracksDeficit(t *testing.T) {
	bus := newTestBus()
	bus.mem[0] = 0x00 // NOP, 4 T-states

	cpu := New(bus)
	if got := cpu.StepCycles(2); got != 2 {
		t.Fatalf("first StepCycles = %d, want 2", got)
	}
	if got := cpu.Deficit(); got != 2 {
		t.Fatalf("deficit = %d, want 2", got)
	}
	if got := cpu.StepCycles(2); got != 2 {
		t.Fatalf("second StepCycles = %d, want 2", got)
	}
	if got := cpu.Deficit(); got != 0 {
		t.Fatalf("deficit = %d, want 0", got)
	}
	if got := cpu.Registers().PC; got != 1 {
		t.Fatalf("PC = %#04x, want %#04x", got, 1)
	}
}

func TestNMI(t *testing.T) {
	bus := newTestBus()
	bus.mem[0] = 0x00 // NOP, should not execute before NMI service.

	cpu := New(bus)
	cpu.NMI()
	if cycles := cpu.Step(); cycles <= 0 {
		t.Fatalf("NMI step returned %d cycles", cycles)
	}

	regs := cpu.Registers()
	if regs.PC != 0x0066 {
		t.Fatalf("PC = %#04x, want NMI vector %#04x", regs.PC, 0x0066)
	}
	if regs.SP != 0xfffd {
		t.Fatalf("SP = %#04x, want %#04x", regs.SP, 0xfffd)
	}
	if bus.mem[0xfffd] != 0x00 || bus.mem[0xfffe] != 0x00 {
		t.Fatalf("pushed PC bytes = %#02x %#02x, want 0 0", bus.mem[0xfffd], bus.mem[0xfffe])
	}
}

type testBus struct {
	mem     [1 << 16]uint8
	in      map[uint16]uint8
	outs    []ioWrite
	fetches int
}

type ioWrite struct {
	port uint16
	val  uint8
}

func newTestBus() *testBus {
	return &testBus{in: make(map[uint16]uint8)}
}

func (b *testBus) Fetch(addr uint16) uint8 {
	b.fetches++
	return b.mem[addr]
}

func (b *testBus) Read(addr uint16) uint8 {
	return b.mem[addr]
}

func (b *testBus) Write(addr uint16, val uint8) {
	b.mem[addr] = val
}

func (b *testBus) In(port uint16) uint8 {
	return b.in[port]
}

func (b *testBus) Out(port uint16, val uint8) {
	b.outs = append(b.outs, ioWrite{port: port, val: val})
}

package cpc

import (
	"testing"

	"cpcgo/internal/keyboard"
	"cpcgo/internal/rom"
)

func TestMachineFetchesFirstInstructionFromLowerROM(t *testing.T) {
	image := testROMImage()
	image.LowerOS[0] = 0x3e // LD A,n
	image.LowerOS[1] = 0x42
	image.LowerOS[2] = 0x76 // HALT

	machine, err := New(Config{Model: Model6128, ROMs: image, Scale: 2})
	if err != nil {
		t.Fatal(err)
	}

	cycles := machine.RunInstructions(2)
	if cycles == 0 {
		t.Fatal("RunInstructions consumed no cycles")
	}
	if got := uint8(machine.CPU().Registers().AF >> 8); got != 0x42 {
		t.Fatalf("A = %#02x, want %#02x", got, 0x42)
	}
	if !machine.CPU().Halted() {
		t.Fatal("CPU did not execute HALT from lower ROM")
	}
}

func TestMachineHardwarePortWrites(t *testing.T) {
	image := testROMImage()
	program := []uint8{
		0x01, 0x00, 0x7f, // LD BC,0x7f00
		0x3e, 0x8e, // LD A,0x8e: mode 2, lower+upper ROM disabled
		0xed, 0x79, // OUT (C),A
		0x3e, 0xc1, // LD A,0xc1: RAM config 1
		0xed, 0x79, // OUT (C),A
		0x01, 0x00, 0xdf, // LD BC,0xdf00
		0x3e, 0x07, // LD A,7: AMSDOS upper ROM bank
		0xed, 0x79, // OUT (C),A
		0x76, // HALT
	}
	copy(image.LowerOS, program)

	machine, err := New(Config{Model: Model6128, ROMs: image, Scale: 2})
	if err != nil {
		t.Fatal(err)
	}
	for offset, val := range program {
		machine.Memory().RAMWrite(0, uint16(offset), val)
	}
	machine.RunInstructions(9)

	if !machine.CPU().Halted() {
		t.Fatal("CPU did not halt after hardware port program")
	}
	if got := machine.GateArray().Mode(); got != 2 {
		t.Fatalf("mode = %d, want 2", got)
	}
	if machine.Memory().LowerROMEnabled() {
		t.Fatal("lower ROM still enabled")
	}
	if machine.Memory().UpperROMEnabled() {
		t.Fatal("upper ROM still enabled")
	}
	if got := machine.Memory().RAMConfig(); got != 1 {
		t.Fatalf("RAM config = %d, want 1", got)
	}
	if got := machine.Memory().SelectedUpperROM(); got != 7 {
		t.Fatalf("selected upper ROM = %d, want 7", got)
	}
}

func TestMachinePPIDrivesPSG(t *testing.T) {
	machine, err := New(Config{Model: Model6128, ROMs: testROMImage(), Scale: 2})
	if err != nil {
		t.Fatal(err)
	}

	machine.IO().Out(0xf400, 7)
	machine.IO().Out(0xf600, 0xc0)
	machine.IO().Out(0xf400, 0x3f)
	machine.IO().Out(0xf600, 0x80)

	if got := machine.PSG().Register(7); got != 0x3f {
		t.Fatalf("PSG register 7 = %#02x, want %#02x", got, 0x3f)
	}
}

func TestMachineKeyboardMatrixFeedsPSGRegister14(t *testing.T) {
	machine, err := New(Config{Model: Model6128, ROMs: testROMImage(), Scale: 2})
	if err != nil {
		t.Fatal(err)
	}

	machine.Keyboard().Press(keyboard.KeyEnter)
	machine.IO().Out(0xf400, 14)
	machine.IO().Out(0xf600, 0xc0)
	machine.IO().Out(0xf600, 0x42)

	if got := machine.PPI().PortA(); got != 0xfb {
		t.Fatalf("keyboard row read = %#02x, want %#02x", got, 0xfb)
	}
}

func TestMachineTimingAdvancesAndUpdatesVSync(t *testing.T) {
	image := testROMImage()
	for i := range image.LowerOS {
		image.LowerOS[i] = 0x00 // NOP
	}
	machine, err := New(Config{Model: Model6128, ROMs: image, Scale: 2})
	if err != nil {
		t.Fatal(err)
	}

	machine.RunInstructions(cyclesPerFrame / 4)

	timing := machine.Timing()
	if timing.Cycles == 0 {
		t.Fatal("timing did not advance")
	}
	if timing.Frames == 0 {
		t.Fatal("timing did not count a frame")
	}
	if timing.Interrupts == 0 {
		t.Fatal("timing did not count interrupts")
	}
	if machine.PPI().PortB()&0x01 != 0 {
		t.Fatal("VSync bit should be clear at start of next frame")
	}
}

func TestMachineTimingCanSetVSync(t *testing.T) {
	image := testROMImage()
	for i := range image.LowerOS {
		image.LowerOS[i] = 0x00 // NOP
	}
	machine, err := New(Config{Model: Model6128, ROMs: image, Scale: 2})
	if err != nil {
		t.Fatal(err)
	}

	machine.RunInstructions((cyclesPerFrame - vsyncCycles) / 4)
	if machine.PPI().PortB()&0x01 == 0 {
		t.Fatal("VSync bit should be set near end of frame")
	}
}

func TestMachineRunFrame(t *testing.T) {
	image := testROMImage()
	for i := range image.LowerOS {
		image.LowerOS[i] = 0x00 // NOP
	}
	machine, err := New(Config{Model: Model6128, ROMs: image, Scale: 2})
	if err != nil {
		t.Fatal(err)
	}

	cycles := machine.RunFrame()
	if cycles == 0 {
		t.Fatal("RunFrame consumed no cycles")
	}
	if got := machine.Timing().Frames; got != 1 {
		t.Fatalf("frames = %d, want 1", got)
	}
}

func TestMachineRejectsInvalidConfig(t *testing.T) {
	_, err := New(Config{Model: "464", ROMs: testROMImage()})
	if err == nil {
		t.Fatal("New accepted unsupported model")
	}

	_, err = New(Config{Model: Model6128, ROMs: rom.Image{LowerOS: make([]byte, rom.BankSize)}})
	if err == nil {
		t.Fatal("New accepted incomplete ROM image")
	}
}

func testROMImage() rom.Image {
	return rom.Image{
		LowerOS: make([]uint8, rom.BankSize),
		Basic:   make([]uint8, rom.BankSize),
		AMSDOS:  make([]uint8, rom.BankSize),
	}
}

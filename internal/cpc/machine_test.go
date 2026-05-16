package cpc

import (
	"testing"

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

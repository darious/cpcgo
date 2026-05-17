package cpc

import (
	"testing"

	"cpcgo/internal/bus"
)

func TestProbeCollectsCPUAndIOStats(t *testing.T) {
	image := testROMImage()
	copy(image.LowerOS, []uint8{
		0x01, 0x00, 0x7f, // LD BC,0x7f00
		0x3e, 0xc1, // LD A,0xc1
		0xed, 0x79, // OUT (C),A
		0x01, 0xff, 0xff, // LD BC,0xffff
		0xed, 0x78, // IN A,(C)
		0x76, // HALT
	})

	machine, err := New(Config{Model: Model6128, ROMs: image})
	if err != nil {
		t.Fatal(err)
	}

	result := machine.Probe(ProbeOptions{Instructions: 6})
	if !result.Halted {
		t.Fatal("probe did not halt")
	}
	if result.Instructions != 6 {
		t.Fatalf("instructions = %d, want 6", result.Instructions)
	}
	if result.Cycles == 0 {
		t.Fatal("probe consumed no cycles")
	}
	if result.Registers.PC == 0 {
		t.Fatal("PC did not advance")
	}
	if result.Timing.Cycles == 0 {
		t.Fatal("probe timing did not advance")
	}
	if result.IO.Writes != 1 {
		t.Fatalf("writes = %d, want 1", result.IO.Writes)
	}
	if result.IO.WritePorts[0x7f00] != 1 {
		t.Fatalf("write ports = %#v, want one write to 0x7f00", result.IO.WritePorts)
	}
	if got := machine.Memory().RAMConfig(); got != 1 {
		t.Fatalf("RAM config = %d, want 1", got)
	}
	if result.IO.Reads != 1 {
		t.Fatalf("reads = %d, want 1", result.IO.Reads)
	}
	if result.IO.UnhandledReads != 1 {
		t.Fatalf("unhandled reads = %d, want 1", result.IO.UnhandledReads)
	}
	if result.IO.UnhandledPorts[0xffff] != 1 {
		t.Fatalf("unhandled ports = %#v, want one read from 0xffff", result.IO.UnhandledPorts)
	}
}

func TestProbeRestoresPreviousIOObserver(t *testing.T) {
	machine, err := New(Config{Model: Model6128, ROMs: testROMImage()})
	if err != nil {
		t.Fatal(err)
	}

	var observed []bus.IOEvent
	machine.IO().SetObserver(func(event bus.IOEvent) {
		observed = append(observed, event)
	})
	_ = machine.Probe(ProbeOptions{Instructions: 1})
	machine.IO().Out(0x1234, 0x56)

	if len(observed) != 1 {
		t.Fatalf("observer event count = %d, want 1", len(observed))
	}
	if observed[0].Port != 0x1234 || observed[0].Value != 0x56 {
		t.Fatalf("observer event = %#v, want write to 0x1234", observed[0])
	}
}

package cpc

import (
	"os"
	"strings"
	"testing"

	"cpcgo/internal/keyboard"
	"cpcgo/internal/rom"
)

func testROMImage() rom.Image {
	return rom.Image{
		LowerOS: make([]byte, rom.BankSize),
		Basic:   make([]byte, rom.BankSize),
		AMSDOS:  make([]byte, rom.BankSize),
	}
}

func newTestMachine(t *testing.T, program []uint8) *Machine {
	t.Helper()
	image := testROMImage()
	copy(image.LowerOS, program)
	machine, err := New(Config{Model: Model6128, ROMs: image})
	if err != nil {
		t.Fatal(err)
	}
	return machine
}

func TestMachineHardwarePortWrites(t *testing.T) {
	program := []uint8{
		0x01, 0x00, 0x7f, // LD BC,0x7f00
		0x3e, 0x8e, // LD A,0x8e: mode 2, lower+upper ROM disabled
		0xed, 0x79, // OUT (C),A
		0x3e, 0xc1, // LD A,0xc1: RAM config 1
		0xed, 0x79, // OUT (C),A
		0x01, 0x00, 0xdf, // LD BC,0xdf00
		0x3e, 0x07, // LD A,7: AMSDOS upper ROM
		0xed, 0x79, // OUT (C),A
		0x76, // HALT
	}
	machine := newTestMachine(t, program)
	// Execution continues from RAM once the lower ROM is disabled.
	for i, b := range program {
		machine.Memory().RAMWrite(0, uint16(i), b)
	}
	machine.RunInstructions(9)

	if got := machine.GateArray().Mode(); got != 2 {
		t.Fatalf("mode = %d, want 2", got)
	}
	if machine.Memory().LowerROMEnabled() || machine.Memory().UpperROMEnabled() {
		t.Fatal("ROMs still enabled")
	}
	if got := machine.Memory().RAMConfig(); got != 1 {
		t.Fatalf("RAM config = %d, want 1", got)
	}
	if got := machine.Memory().SelectedUpperROM(); got != 7 {
		t.Fatalf("upper ROM = %d, want 7", got)
	}
	if !machine.CPU().Halted() {
		t.Fatal("CPU did not halt")
	}
}

// With the CRTC at its defaults (64x312 characters) the Gate Array raises six
// interrupts per 19968-microsecond frame and the monitor completes one frame.
func TestMachineInterruptAndFrameCadence(t *testing.T) {
	// IM 1; EI; JR $ with an EI; RET handler at 0x0038.
	image := testROMImage()
	copy(image.LowerOS, []uint8{0xed, 0x56, 0xfb, 0x18, 0xfe})
	image.LowerOS[0x38], image.LowerOS[0x39] = 0xfb, 0xc9
	machine, err := New(Config{Model: Model6128, ROMs: image})
	if err != nil {
		t.Fatal(err)
	}

	interrupts := 0
	machine.RunFrame() // let the monitor lock to VSYNC
	start := machine.Micros()
	frames := machine.Monitor().Frames()
	for machine.Micros()-start < 19968*10 {
		pc := machine.CPU().Registers().PC
		machine.Step()
		if machine.CPU().Registers().PC == 0x38 && pc != 0x38 {
			interrupts++
		}
	}
	if got := machine.Monitor().Frames() - frames; got != 10 {
		t.Fatalf("frames in 10 frame periods = %d, want 10", got)
	}
	if interrupts < 59 || interrupts > 61 {
		t.Fatalf("interrupts in 10 frames = %d, want 60", interrupts)
	}
}

func TestMachineReadsKeyboardThroughPSG(t *testing.T) {
	machine := newTestMachine(t, []uint8{
		0x01, 0x82, 0xf7, // LD BC,0xf782: PPI port A output, C output
		0xed, 0x49, // OUT (C),C
		0x01, 0x0e, 0xf4, // LD BC,0xf40e: PSG register 14
		0xed, 0x49, // OUT (C),C
		0x01, 0xc0, 0xf6, // LD BC,0xf6c0: latch address
		0xed, 0x49, // OUT (C),C
		0x01, 0x00, 0xf6, // LD BC,0xf600: inactive
		0xed, 0x49, // OUT (C),C
		0x01, 0x92, 0xf7, // LD BC,0xf792: port A input
		0xed, 0x49, // OUT (C),C
		0x01, 0x48, 0xf6, // LD BC,0xf648: PSG read, keyboard line 8
		0xed, 0x49, // OUT (C),C
		0x06, 0xf4, // LD B,0xf4
		0xed, 0x78, // IN A,(C)
		0x76, // HALT
	})
	machine.Keyboard().Press(keyboard.KeyA)
	machine.RunInstructions(16)
	if got := uint8(machine.CPU().Registers().AF >> 8); got != 0xdf {
		t.Fatalf("keyboard line 8 = %#02x, want 0xdf (A pressed)", got)
	}
}

func TestMachineRejectsInvalidConfig(t *testing.T) {
	if _, err := New(Config{Model: "999", ROMs: testROMImage()}); err == nil {
		t.Fatal("unsupported model accepted")
	}
	if _, err := New(Config{Model: Model6128, ROMs: rom.Image{}}); err == nil {
		t.Fatal("missing ROMs accepted")
	}
}

// TestBootsToBASIC runs the real firmware when the ROMs are present in the
// repository root (they are not committed).
func TestBootsToBASIC(t *testing.T) {
	for _, tc := range []struct {
		model  Model
		file   string
		banner string
	}{
		{Model464, "cpc464.rom", "BASIC 1.0"},
		{Model664, "cpc664.rom", "BASIC 1.1"},
		{Model6128, "cpc6128.rom", "BASIC 1.1"},
	} {
		image, err := rom.LoadOSBasic("../../" + tc.file)
		if err != nil {
			t.Skipf("ROM not available: %v", err)
		}
		if image, err = image.LoadAMSDOS("../../amsdos.rom"); err != nil {
			t.Skipf("AMSDOS ROM not available: %v", err)
		}
		machine, err := New(Config{Model: tc.model, ROMs: image})
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 150; i++ {
			machine.RunFrame()
		}
		text := screenText(machine)
		if !strings.Contains(text, tc.banner) || !strings.Contains(text, "Ready") {
			t.Errorf("%s screen does not show %q and Ready:\n%s", tc.model, tc.banner, text)
		}
	}
}

// screenText decodes the mode 1 screen at &C000 using the font in the lower
// ROM (character matrix at &3800).
func screenText(m *Machine) string {
	if _, err := os.Stat("../../cpc6128.rom"); err != nil {
		return ""
	}
	font := make(map[[8]uint8]byte)
	for ch := 32; ch < 127; ch++ {
		var glyph [8]uint8
		for row := 0; row < 8; row++ {
			glyph[row] = m.memory.LowerROM()[0x3800+uint16(ch)*8+uint16(row)]
		}
		font[glyph] = byte(ch)
	}
	var out strings.Builder
	for row := 0; row < 25; row++ {
		for col := 0; col < 40; col++ {
			var glyph [8]uint8
			for line := 0; line < 8; line++ {
				addr := uint16(0xc000) + uint16(line)*0x800 + uint16(row*80+col*2)
				hi, lo := m.memory.VideoRead(addr), m.memory.VideoRead(addr+1)
				glyph[line] = mode1Mask(hi)<<4 | mode1Mask(lo)
			}
			if ch, ok := font[glyph]; ok {
				out.WriteByte(ch)
			} else {
				out.WriteByte('?')
			}
		}
		out.WriteByte('\n')
	}
	return out.String()
}

// mode1Mask returns a 4-bit mask of the non-zero pixels in a mode 1 byte.
func mode1Mask(b uint8) uint8 {
	return (b | b<<4) >> 4
}

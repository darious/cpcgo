package rom

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOSBasicSplitsBanks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpc6128.rom")
	data := make([]byte, OSBasicSize)
	data[0] = 0x11
	data[BankSize] = 0x22
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	image, err := LoadOSBasic(path)
	if err != nil {
		t.Fatal(err)
	}

	if got := len(image.LowerOS); got != BankSize {
		t.Fatalf("lower OS size = %d, want %d", got, BankSize)
	}
	if got := len(image.Basic); got != BankSize {
		t.Fatalf("BASIC size = %d, want %d", got, BankSize)
	}
	if image.LowerOS[0] != 0x11 {
		t.Fatalf("lower OS first byte = %#x, want %#x", image.LowerOS[0], 0x11)
	}
	if image.Basic[0] != 0x22 {
		t.Fatalf("BASIC first byte = %#x, want %#x", image.Basic[0], 0x22)
	}
}

func TestLoadOSBasicRejectsWrongSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.rom")
	if err := os.WriteFile(path, make([]byte, BankSize), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadOSBasic(path)
	if err == nil {
		t.Fatal("LoadOSBasic succeeded for wrong-sized ROM")
	}
}

func TestLoadAMSDOS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "amsdos.rom")
	data := make([]byte, ExpansionSize)
	data[0] = 0x33
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	image, err := (Image{}).LoadAMSDOS(path)
	if err != nil {
		t.Fatal(err)
	}

	if got := len(image.AMSDOS); got != ExpansionSize {
		t.Fatalf("AMSDOS size = %d, want %d", got, ExpansionSize)
	}
	if image.AMSDOS[0] != 0x33 {
		t.Fatalf("AMSDOS first byte = %#x, want %#x", image.AMSDOS[0], 0x33)
	}
}

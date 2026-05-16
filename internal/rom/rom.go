// Package rom loads and validates CPC ROM images.
package rom

import (
	"fmt"
	"os"
)

const (
	BankSize      = 16 * 1024
	OSBasicSize   = 2 * BankSize
	ExpansionSize = BankSize
)

// Image contains the ROM banks used by a CPC6128 machine.
type Image struct {
	LowerOS []byte
	Basic   []byte
	AMSDOS  []byte
}

// LoadOSBasic reads a 32K CPC OS+BASIC ROM image and splits it into two 16K
// banks.
func LoadOSBasic(path string) (Image, error) {
	data, err := readExact(path, OSBasicSize)
	if err != nil {
		return Image{}, err
	}

	return Image{
		LowerOS: cloneBank(data[:BankSize]),
		Basic:   cloneBank(data[BankSize:OSBasicSize]),
	}, nil
}

// LoadAMSDOS reads a 16K AMSDOS expansion ROM into the image.
func (i Image) LoadAMSDOS(path string) (Image, error) {
	data, err := readExact(path, ExpansionSize)
	if err != nil {
		return Image{}, err
	}

	i.AMSDOS = cloneBank(data)
	return i, nil
}

func readExact(path string, want int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) != want {
		return nil, fmt.Errorf("%s: got %d bytes, want %d", path, len(data), want)
	}
	return data, nil
}

func cloneBank(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)
	return out
}

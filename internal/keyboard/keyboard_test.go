package keyboard

import "testing"

func TestMatrixDefaultsHigh(t *testing.T) {
	matrix := New()
	for line := uint8(0); line < LineCount; line++ {
		if got := matrix.Row(line); got != 0xff {
			t.Fatalf("line %d = %#02x, want %#02x", line, got, 0xff)
		}
	}
	if got := matrix.Row(15); got != 0xff {
		t.Fatalf("invalid line = %#02x, want %#02x", got, 0xff)
	}
}

func TestMatrixPressRelease(t *testing.T) {
	matrix := New()

	matrix.Press(KeyEnter)
	if got := matrix.Row(KeyEnter.Line); got != 0xfb {
		t.Fatalf("enter row = %#02x, want %#02x", got, 0xfb)
	}

	matrix.Release(KeyEnter)
	if got := matrix.Row(KeyEnter.Line); got != 0xff {
		t.Fatalf("enter row after release = %#02x, want %#02x", got, 0xff)
	}
}

func TestMatrixIgnoresInvalidKeys(t *testing.T) {
	matrix := New()
	matrix.Press(Key{Line: LineCount, Bit: 0})
	matrix.Press(Key{Line: 0, Bit: BitCount})
	if got := matrix.Row(0); got != 0xff {
		t.Fatalf("line 0 = %#02x, want unchanged %#02x", got, 0xff)
	}
}

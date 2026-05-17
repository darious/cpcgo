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

	matrix.Press(KeyReturn)
	if got := matrix.Row(KeyReturn.Line); got != 0xfb {
		t.Fatalf("return row = %#02x, want %#02x", got, 0xfb)
	}

	matrix.Release(KeyReturn)
	if got := matrix.Row(KeyReturn.Line); got != 0xff {
		t.Fatalf("return row after release = %#02x, want %#02x", got, 0xff)
	}
}

func TestCPCMatrixCoordinates(t *testing.T) {
	tests := map[string]Key{
		"P":      {Line: 3, Bit: 3},
		"R":      {Line: 6, Bit: 2},
		"I":      {Line: 4, Bit: 3},
		"N":      {Line: 5, Bit: 6},
		"T":      {Line: 6, Bit: 3},
		"1":      {Line: 8, Bit: 0},
		"shift":  {Line: 2, Bit: 5},
		"return": {Line: 2, Bit: 2},
		"space":  {Line: 5, Bit: 7},
		"z":      {Line: 8, Bit: 7},
	}

	for name, want := range tests {
		key := map[string]Key{
			"P":      KeyP,
			"R":      KeyR,
			"I":      KeyI,
			"N":      KeyN,
			"T":      KeyT,
			"1":      Key1,
			"shift":  KeyShift,
			"return": KeyReturn,
			"space":  KeySpace,
			"z":      KeyZ,
		}[name]
		if key != want {
			t.Fatalf("%s = %#v, want %#v", name, key, want)
		}
	}
}

func TestSequenceForText(t *testing.T) {
	sequence, err := SequenceForText("PRINT 1+1\n")
	if err != nil {
		t.Fatal(err)
	}

	want := []Chord{
		{KeyP},
		{KeyR},
		{KeyI},
		{KeyN},
		{KeyT},
		{KeySpace},
		{Key1},
		{KeyShift, KeySemicolon},
		{Key1},
		{KeyReturn},
	}
	if len(sequence) != len(want) {
		t.Fatalf("sequence length = %d, want %d", len(sequence), len(want))
	}
	for i := range sequence {
		if !sameChord(sequence[i], want[i]) {
			t.Fatalf("sequence[%d] = %#v, want %#v", i, sequence[i], want[i])
		}
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

func sameChord(a Chord, b Chord) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

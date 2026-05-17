// Package keyboard maps input to the CPC keyboard matrix.
package keyboard

const (
	LineCount = 10
	BitCount  = 8
)

// Key identifies a CPC keyboard matrix position.
type Key struct {
	Line uint8
	Bit  uint8
}

var (
	KeyEnter = Key{Line: 2, Bit: 2}
	KeySpace = Key{Line: 5, Bit: 7}
	KeyShift = Key{Line: 2, Bit: 5}
	KeyCtrl  = Key{Line: 2, Bit: 7}
	KeyP     = Key{Line: 3, Bit: 3}
	KeyR     = Key{Line: 2, Bit: 3}
	KeyI     = Key{Line: 4, Bit: 3}
	KeyN     = Key{Line: 6, Bit: 2}
	KeyT     = Key{Line: 2, Bit: 4}
	Key1     = Key{Line: 8, Bit: 0}
	KeyPlus  = Key{Line: 3, Bit: 1}
)

// Matrix stores active-low CPC keyboard rows. A cleared bit means pressed.
type Matrix struct {
	lines [LineCount]uint8
}

// New creates a keyboard matrix with all keys released.
func New() *Matrix {
	m := &Matrix{}
	m.ReleaseAll()
	return m
}

// ReleaseAll releases every key.
func (m *Matrix) ReleaseAll() {
	for i := range m.lines {
		m.lines[i] = 0xff
	}
}

// Set sets a key pressed/released state.
func (m *Matrix) Set(key Key, pressed bool) {
	if key.Line >= LineCount || key.Bit >= BitCount {
		return
	}
	mask := uint8(1 << key.Bit)
	if pressed {
		m.lines[key.Line] &^= mask
		return
	}
	m.lines[key.Line] |= mask
}

// Press presses a key.
func (m *Matrix) Press(key Key) {
	m.Set(key, true)
}

// Release releases a key.
func (m *Matrix) Release(key Key) {
	m.Set(key, false)
}

// Row returns the active-low row byte for a keyboard line.
func (m *Matrix) Row(line uint8) uint8 {
	if line >= LineCount {
		return 0xff
	}
	return m.lines[line]
}

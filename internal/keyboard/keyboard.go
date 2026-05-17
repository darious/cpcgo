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
	KeyCursorUp    = Key{Line: 0, Bit: 0}
	KeyCursorRight = Key{Line: 0, Bit: 1}
	KeyCursorDown  = Key{Line: 0, Bit: 2}
	KeyF9          = Key{Line: 0, Bit: 3}
	KeyF6          = Key{Line: 0, Bit: 4}
	KeyF3          = Key{Line: 0, Bit: 5}
	KeyEnter       = Key{Line: 0, Bit: 6}
	KeyNumpadDot   = Key{Line: 0, Bit: 7}

	KeyCursorLeft = Key{Line: 1, Bit: 0}
	KeyCopy       = Key{Line: 1, Bit: 1}
	KeyF7         = Key{Line: 1, Bit: 2}
	KeyF8         = Key{Line: 1, Bit: 3}
	KeyF5         = Key{Line: 1, Bit: 4}
	KeyF1         = Key{Line: 1, Bit: 5}
	KeyF2         = Key{Line: 1, Bit: 6}
	KeyF0         = Key{Line: 1, Bit: 7}

	KeyClr          = Key{Line: 2, Bit: 0}
	KeyLeftBrace    = Key{Line: 2, Bit: 1}
	KeyReturn       = Key{Line: 2, Bit: 2}
	KeyRightBrace   = Key{Line: 2, Bit: 3}
	KeyF4           = Key{Line: 2, Bit: 4}
	KeyShift        = Key{Line: 2, Bit: 5}
	KeyBackslash    = Key{Line: 2, Bit: 6}
	KeyCtrl         = Key{Line: 2, Bit: 7}
	KeyBackspace    = KeyClr
	KeyMainEnter    = KeyReturn
	KeyOpenBracket  = KeyLeftBrace
	KeyCloseBracket = KeyRightBrace
	KeyBackquote    = KeyBackslash

	KeyCaret        = Key{Line: 3, Bit: 0}
	KeyHyphen       = Key{Line: 3, Bit: 1}
	KeyAt           = Key{Line: 3, Bit: 2}
	KeyP            = Key{Line: 3, Bit: 3}
	KeySemicolon    = Key{Line: 3, Bit: 4}
	KeyColon        = Key{Line: 3, Bit: 5}
	KeySlash        = Key{Line: 3, Bit: 6}
	KeyComma        = Key{Line: 3, Bit: 7}
	KeyMinus        = KeyHyphen
	KeyEqual        = KeyHyphen
	KeyPlus         = KeySemicolon
	KeyAsterisk     = KeyColon
	KeyLeftBracket  = KeyLeftBrace
	KeyRightBracket = KeyRightBrace
	KeyPipe         = KeyAt
	KeyGreater      = KeyComma
	KeyLess         = KeyPeriod

	Key0      = Key{Line: 4, Bit: 0}
	Key9      = Key{Line: 4, Bit: 1}
	KeyO      = Key{Line: 4, Bit: 2}
	KeyI      = Key{Line: 4, Bit: 3}
	KeyL      = Key{Line: 4, Bit: 4}
	KeyK      = Key{Line: 4, Bit: 5}
	KeyM      = Key{Line: 4, Bit: 6}
	KeyPeriod = Key{Line: 4, Bit: 7}

	Key8     = Key{Line: 5, Bit: 0}
	Key7     = Key{Line: 5, Bit: 1}
	KeyU     = Key{Line: 5, Bit: 2}
	KeyY     = Key{Line: 5, Bit: 3}
	KeyH     = Key{Line: 5, Bit: 4}
	KeyJ     = Key{Line: 5, Bit: 5}
	KeyN     = Key{Line: 5, Bit: 6}
	KeySpace = Key{Line: 5, Bit: 7}

	Key6 = Key{Line: 6, Bit: 0}
	Key5 = Key{Line: 6, Bit: 1}
	KeyR = Key{Line: 6, Bit: 2}
	KeyT = Key{Line: 6, Bit: 3}
	KeyG = Key{Line: 6, Bit: 4}
	KeyF = Key{Line: 6, Bit: 5}
	KeyB = Key{Line: 6, Bit: 6}
	KeyV = Key{Line: 6, Bit: 7}

	Key4 = Key{Line: 7, Bit: 0}
	Key3 = Key{Line: 7, Bit: 1}
	KeyE = Key{Line: 7, Bit: 2}
	KeyW = Key{Line: 7, Bit: 3}
	KeyS = Key{Line: 7, Bit: 4}
	KeyD = Key{Line: 7, Bit: 5}
	KeyC = Key{Line: 7, Bit: 6}
	KeyX = Key{Line: 7, Bit: 7}

	Key1        = Key{Line: 8, Bit: 0}
	Key2        = Key{Line: 8, Bit: 1}
	KeyEsc      = Key{Line: 8, Bit: 2}
	KeyQ        = Key{Line: 8, Bit: 3}
	KeyTab      = Key{Line: 8, Bit: 4}
	KeyA        = Key{Line: 8, Bit: 5}
	KeyCapsLock = Key{Line: 8, Bit: 6}
	KeyZ        = Key{Line: 8, Bit: 7}
	KeyEscape   = KeyEsc

	KeyJoy0Up    = Key{Line: 9, Bit: 0}
	KeyJoy0Down  = Key{Line: 9, Bit: 1}
	KeyJoy0Left  = Key{Line: 9, Bit: 2}
	KeyJoy0Right = Key{Line: 9, Bit: 3}
	KeyJoy0Fire1 = Key{Line: 9, Bit: 4}
	KeyJoy0Fire2 = Key{Line: 9, Bit: 5}
	KeyDel       = Key{Line: 9, Bit: 7}
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

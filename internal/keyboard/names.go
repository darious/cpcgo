package keyboard

// Names maps the key names used by cpc-validation input scripts
// (key_press/key_release) to matrix positions.
var Names = map[string]Key{
	"ArrowUp": KeyCursorUp, "ArrowRight": KeyCursorRight, "ArrowDown": KeyCursorDown,
	"ArrowLeft": KeyCursorLeft, "Copy": KeyCopy,
	"Numpad0": KeyF0, "Numpad1": KeyF1, "Numpad2": KeyF2, "Numpad3": KeyF3, "Numpad4": KeyF4,
	"Numpad5": KeyF5, "Numpad6": KeyF6, "Numpad7": KeyF7, "Numpad8": KeyF8, "Numpad9": KeyF9,
	"NumpadEnter": KeyEnter, "NumpadPeriod": KeyNumpadDot,
	"Clear": KeyClr, "BracketLeft": KeyLeftBrace, "Enter": KeyReturn, "BracketRight": KeyRightBrace,
	"Shift": KeyShift, "Backslash": KeyBackslash, "Control": KeyCtrl,
	"Caret": KeyCaret, "Minus": KeyHyphen, "At": KeyAt, "Semicolon": KeySemicolon,
	"Colon": KeyColon, "Slash": KeySlash, "Period": KeyPeriod, "Comma": KeyComma,
	"Space": KeySpace, "Escape": KeyEsc, "Tab": KeyTab, "CapsLock": KeyCapsLock, "Delete": KeyDel,
	"Key0": Key0, "Key1": Key1, "Key2": Key2, "Key3": Key3, "Key4": Key4,
	"Key5": Key5, "Key6": Key6, "Key7": Key7, "Key8": Key8, "Key9": Key9,
	"A": KeyA, "B": KeyB, "C": KeyC, "D": KeyD, "E": KeyE, "F": KeyF, "G": KeyG,
	"H": KeyH, "I": KeyI, "J": KeyJ, "K": KeyK, "L": KeyL, "M": KeyM, "N": KeyN,
	"O": KeyO, "P": KeyP, "Q": KeyQ, "R": KeyR, "S": KeyS, "T": KeyT, "U": KeyU,
	"V": KeyV, "W": KeyW, "X": KeyX, "Y": KeyY, "Z": KeyZ,
	"JoystickUp": KeyJoy0Up, "JoystickDown": KeyJoy0Down, "JoystickLeft": KeyJoy0Left,
	"JoystickRight": KeyJoy0Right, "JoystickFire1": KeyJoy0Fire1, "JoystickFire2": KeyJoy0Fire2,
	"JoystickFire3": {Line: 9, Bit: 6},
}

// ChordForRune returns the keys to press to type r, if it can be typed.
func ChordForRune(r rune) (Chord, bool) {
	return chordForRune(r)
}

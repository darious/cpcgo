package keyboard

import "fmt"

// Chord is a set of CPC keys that should be pressed together for one input
// step.
type Chord []Key

// SequenceForText converts a small BASIC-oriented text string to CPC key
// chords. It is intended for smoke tests and later scripted input, not as a
// replacement for the live keyboard matrix.
func SequenceForText(text string) ([]Chord, error) {
	sequence := make([]Chord, 0, len(text))
	for _, r := range text {
		chord, ok := chordForRune(r)
		if !ok {
			return nil, fmt.Errorf("unsupported CPC scripted input rune %q", r)
		}
		sequence = append(sequence, chord)
	}
	return sequence, nil
}

func chordForRune(r rune) (Chord, bool) {
	if key, ok := unshiftedRunes[r]; ok {
		return Chord{key}, true
	}
	if key, ok := shiftedRunes[r]; ok {
		return Chord{KeyShift, key}, true
	}
	return nil, false
}

var unshiftedRunes = map[rune]Key{
	'\n': KeyReturn,
	'\r': KeyReturn,
	' ':  KeySpace,
	'0':  Key0,
	'1':  Key1,
	'2':  Key2,
	'3':  Key3,
	'4':  Key4,
	'5':  Key5,
	'6':  Key6,
	'7':  Key7,
	'8':  Key8,
	'9':  Key9,
	'-':  KeyMinus,
	'.':  KeyPeriod,
	',':  KeyComma,
	'/':  KeySlash,
	';':  KeySemicolon,
	':':  KeyColon,
	'@':  KeyAt,
	'[':  KeyLeftBrace,
	']':  KeyRightBrace,
	'\\': KeyBackslash,
	'^':  KeyCaret,
}

var shiftedRunes = map[rune]Key{
	'!':  Key1,
	'"':  Key2,
	'#':  Key3,
	'$':  Key4,
	'%':  Key5,
	'&':  Key6,
	'\'': Key7,
	'(':  Key8,
	')':  Key9,
	'_':  Key0,
	'=':  KeyHyphen,
	'+':  KeySemicolon,
	'*':  KeyColon,
	'?':  KeySlash,
	'>':  KeyComma,
	'<':  KeyPeriod,
	'|':  KeyAt,
	'`':  KeyBackquote,
	'{':  KeyLeftBrace,
	'}':  KeyRightBrace,
}

func init() {
	for r := 'A'; r <= 'Z'; r++ {
		unshiftedRunes[r] = letterKeys[r]
	}
	for r := 'a'; r <= 'z'; r++ {
		unshiftedRunes[r] = letterKeys[r-'a'+'A']
	}
}

var letterKeys = map[rune]Key{
	'A': KeyA,
	'B': KeyB,
	'C': KeyC,
	'D': KeyD,
	'E': KeyE,
	'F': KeyF,
	'G': KeyG,
	'H': KeyH,
	'I': KeyI,
	'J': KeyJ,
	'K': KeyK,
	'L': KeyL,
	'M': KeyM,
	'N': KeyN,
	'O': KeyO,
	'P': KeyP,
	'Q': KeyQ,
	'R': KeyR,
	'S': KeyS,
	'T': KeyT,
	'U': KeyU,
	'V': KeyV,
	'W': KeyW,
	'X': KeyX,
	'Y': KeyY,
	'Z': KeyZ,
}

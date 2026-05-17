//go:build liveui

package ebitenui

import (
	"testing"

	"cpcgo/internal/keyboard"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDefaultKeyMapCoversBasicInput(t *testing.T) {
	keys := defaultKeyMap()

	tests := map[ebiten.Key]keyboard.Chord{
		ebiten.KeyP:           {keyboard.KeyP},
		ebiten.KeyR:           {keyboard.KeyR},
		ebiten.KeyI:           {keyboard.KeyI},
		ebiten.KeyN:           {keyboard.KeyN},
		ebiten.KeyT:           {keyboard.KeyT},
		ebiten.KeySpace:       {keyboard.KeySpace},
		ebiten.KeyDigit1:      {keyboard.Key1},
		ebiten.KeyNumpadAdd:   {keyboard.KeyShift, keyboard.KeySemicolon},
		ebiten.KeyEnter:       {keyboard.KeyReturn},
		ebiten.KeyArrowLeft:   {keyboard.KeyCursorLeft},
		ebiten.KeyBackspace:   {keyboard.KeyClr},
		ebiten.KeyControlLeft: {keyboard.KeyCtrl},
	}

	for host, want := range tests {
		got, ok := keys[host]
		if !ok {
			t.Fatalf("host key %s is not mapped", host)
		}
		if !sameChord(got, want) {
			t.Fatalf("host key %s = %#v, want %#v", host, got, want)
		}
	}
}

func TestLayoutUsesLineDoubledCPCAspect(t *testing.T) {
	game := &game{}
	width, height := game.Layout(0, 0)
	if width != screenWidth || height != screenHeight {
		t.Fatalf("layout = %dx%d, want %dx%d", width, height, screenWidth, screenHeight)
	}
	if activeY+videoHeight()*pixelYScale > height {
		t.Fatalf("active display exceeds layout height")
	}
}

func sameChord(a keyboard.Chord, b keyboard.Chord) bool {
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

func videoHeight() int {
	return 200
}

//go:build liveui

// Package ebitenui provides the live Ebiten frontend.
package ebitenui

import (
	"image/color"

	"cpcgo/internal/cpc"
	"cpcgo/internal/keyboard"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 640
	screenHeight = 200
)

// Config configures the live UI.
type Config struct {
	Scale int
}

// Run starts the Ebiten live emulator UI.
func Run(machine *cpc.Machine, config Config) error {
	if config.Scale < 1 {
		config.Scale = 2
	}

	ebiten.SetWindowTitle("cpcgo")
	ebiten.SetWindowSize(screenWidth*config.Scale, screenHeight*config.Scale)
	ebiten.SetTPS(cpc.FramesPerSecond)

	return ebiten.RunGame(&game{
		machine: machine,
		scale:   config.Scale,
		keys:    defaultKeyMap(),
	})
}

type game struct {
	machine *cpc.Machine
	scale   int
	keys    map[ebiten.Key]keyboard.Key
	frame   *ebiten.Image
}

func (g *game) Update() error {
	g.updateKeyboard()
	g.machine.RunFrame()
	g.frame = ebiten.NewImageFromImage(g.machine.Framebuffer())
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	if g.frame == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(g.scale), float64(g.scale))
	screen.DrawImage(g.frame, op)
}

func (g *game) Layout(int, int) (int, int) {
	return screenWidth * g.scale, screenHeight * g.scale
}

func (g *game) updateKeyboard() {
	matrix := g.machine.Keyboard()
	for host, cpcKey := range g.keys {
		matrix.Set(cpcKey, ebiten.IsKeyPressed(host))
	}
}

func defaultKeyMap() map[ebiten.Key]keyboard.Key {
	return map[ebiten.Key]keyboard.Key{
		ebiten.KeyEnter:        keyboard.KeyEnter,
		ebiten.KeySpace:        keyboard.KeySpace,
		ebiten.KeyShift:        keyboard.KeyShift,
		ebiten.KeyControl:      keyboard.KeyCtrl,
		ebiten.KeyP:            keyboard.KeyP,
		ebiten.KeyR:            keyboard.KeyR,
		ebiten.KeyI:            keyboard.KeyI,
		ebiten.KeyN:            keyboard.KeyN,
		ebiten.KeyT:            keyboard.KeyT,
		ebiten.KeyDigit1:       keyboard.Key1,
		ebiten.KeyNumpad1:      keyboard.Key1,
		ebiten.KeyEqual:        keyboard.KeyPlus,
		ebiten.KeyNumpadAdd:    keyboard.KeyPlus,
		ebiten.KeyShiftLeft:    keyboard.KeyShift,
		ebiten.KeyShiftRight:   keyboard.KeyShift,
		ebiten.KeyControlLeft:  keyboard.KeyCtrl,
		ebiten.KeyControlRight: keyboard.KeyCtrl,
	}
}

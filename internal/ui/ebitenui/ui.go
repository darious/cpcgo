//go:build liveui

// Package ebitenui provides the live Ebiten frontend.
package ebitenui

import (
	"image"
	"image/png"
	"os"

	"cpcgo/internal/cpc"
	"cpcgo/internal/keyboard"
	"cpcgo/internal/video"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 768
	screenHeight = 544
	activeX      = 64
	activeY      = 72
	pixelYScale  = 2
)

// Config configures the live UI.
type Config struct {
	Scale          int
	ScreenshotPath string
}

// Run starts the Ebiten live emulator UI.
func Run(machine *cpc.Machine, config Config) error {
	if config.Scale < 1 {
		config.Scale = 1
	}

	ebiten.SetWindowTitle("cpcgo")
	ebiten.SetWindowSize(screenWidth*config.Scale, screenHeight*config.Scale)
	ebiten.SetTPS(cpc.FramesPerSecond)

	return ebiten.RunGame(&game{
		machine:        machine,
		keys:           defaultKeyMap(),
		screenshotPath: config.ScreenshotPath,
	})
}

type game struct {
	machine *cpc.Machine
	keys    map[ebiten.Key]keyboard.Chord
	frame   *ebiten.Image

	screenshotPath string
}

func (g *game) Update() error {
	g.updateKeyboard()
	g.machine.RunFrame()
	g.frame = ebiten.NewImageFromImage(g.machine.Framebuffer())
	if g.screenshotPath != "" && inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		return saveSnapshot(g.machine, g.screenshotPath)
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(video.HardwareColor(g.machine.BorderInk()))
	if g.frame == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1, pixelYScale)
	op.GeoM.Translate(activeX, activeY)
	screen.DrawImage(g.frame, op)
}

func (g *game) Layout(int, int) (int, int) {
	return screenWidth, screenHeight
}

func (g *game) updateKeyboard() {
	matrix := g.machine.Keyboard()
	matrix.ReleaseAll()
	for host, chord := range g.keys {
		if !ebiten.IsKeyPressed(host) {
			continue
		}
		for _, cpcKey := range chord {
			matrix.Press(cpcKey)
		}
	}
}

func saveSnapshot(machine *cpc.Machine, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, snapshot(machine))
}

func snapshot(machine *cpc.Machine) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	border := video.HardwareColor(machine.BorderInk())
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			img.SetRGBA(x, y, border)
		}
	}

	frame := machine.Framebuffer()
	for y := 0; y < video.Height; y++ {
		for x := 0; x < video.Width; x++ {
			c := frame.At(x, y)
			img.Set(activeX+x, activeY+y*pixelYScale, c)
			img.Set(activeX+x, activeY+y*pixelYScale+1, c)
		}
	}
	return img
}

func defaultKeyMap() map[ebiten.Key]keyboard.Chord {
	return map[ebiten.Key]keyboard.Chord{
		ebiten.KeyA:              {keyboard.KeyA},
		ebiten.KeyB:              {keyboard.KeyB},
		ebiten.KeyC:              {keyboard.KeyC},
		ebiten.KeyD:              {keyboard.KeyD},
		ebiten.KeyE:              {keyboard.KeyE},
		ebiten.KeyF:              {keyboard.KeyF},
		ebiten.KeyG:              {keyboard.KeyG},
		ebiten.KeyH:              {keyboard.KeyH},
		ebiten.KeyI:              {keyboard.KeyI},
		ebiten.KeyJ:              {keyboard.KeyJ},
		ebiten.KeyK:              {keyboard.KeyK},
		ebiten.KeyL:              {keyboard.KeyL},
		ebiten.KeyM:              {keyboard.KeyM},
		ebiten.KeyN:              {keyboard.KeyN},
		ebiten.KeyO:              {keyboard.KeyO},
		ebiten.KeyP:              {keyboard.KeyP},
		ebiten.KeyQ:              {keyboard.KeyQ},
		ebiten.KeyR:              {keyboard.KeyR},
		ebiten.KeyS:              {keyboard.KeyS},
		ebiten.KeyT:              {keyboard.KeyT},
		ebiten.KeyU:              {keyboard.KeyU},
		ebiten.KeyV:              {keyboard.KeyV},
		ebiten.KeyW:              {keyboard.KeyW},
		ebiten.KeyX:              {keyboard.KeyX},
		ebiten.KeyY:              {keyboard.KeyY},
		ebiten.KeyZ:              {keyboard.KeyZ},
		ebiten.KeyDigit0:         {keyboard.Key0},
		ebiten.KeyDigit1:         {keyboard.Key1},
		ebiten.KeyDigit2:         {keyboard.Key2},
		ebiten.KeyDigit3:         {keyboard.Key3},
		ebiten.KeyDigit4:         {keyboard.Key4},
		ebiten.KeyDigit5:         {keyboard.Key5},
		ebiten.KeyDigit6:         {keyboard.Key6},
		ebiten.KeyDigit7:         {keyboard.Key7},
		ebiten.KeyDigit8:         {keyboard.Key8},
		ebiten.KeyDigit9:         {keyboard.Key9},
		ebiten.KeyNumpad0:        {keyboard.Key0},
		ebiten.KeyNumpad1:        {keyboard.Key1},
		ebiten.KeyNumpad2:        {keyboard.Key2},
		ebiten.KeyNumpad3:        {keyboard.Key3},
		ebiten.KeyNumpad4:        {keyboard.Key4},
		ebiten.KeyNumpad5:        {keyboard.Key5},
		ebiten.KeyNumpad6:        {keyboard.Key6},
		ebiten.KeyNumpad7:        {keyboard.Key7},
		ebiten.KeyNumpad8:        {keyboard.Key8},
		ebiten.KeyNumpad9:        {keyboard.Key9},
		ebiten.KeyEnter:          {keyboard.KeyReturn},
		ebiten.KeyNumpadEnter:    {keyboard.KeyReturn},
		ebiten.KeySpace:          {keyboard.KeySpace},
		ebiten.KeyTab:            {keyboard.KeyTab},
		ebiten.KeyEscape:         {keyboard.KeyEsc},
		ebiten.KeyBackspace:      {keyboard.KeyClr},
		ebiten.KeyDelete:         {keyboard.KeyDel},
		ebiten.KeyArrowUp:        {keyboard.KeyCursorUp},
		ebiten.KeyArrowRight:     {keyboard.KeyCursorRight},
		ebiten.KeyArrowDown:      {keyboard.KeyCursorDown},
		ebiten.KeyArrowLeft:      {keyboard.KeyCursorLeft},
		ebiten.KeyShift:          {keyboard.KeyShift},
		ebiten.KeyShiftLeft:      {keyboard.KeyShift},
		ebiten.KeyShiftRight:     {keyboard.KeyShift},
		ebiten.KeyControl:        {keyboard.KeyCtrl},
		ebiten.KeyControlLeft:    {keyboard.KeyCtrl},
		ebiten.KeyControlRight:   {keyboard.KeyCtrl},
		ebiten.KeyMinus:          {keyboard.KeyHyphen},
		ebiten.KeyEqual:          {keyboard.KeySemicolon},
		ebiten.KeyNumpadAdd:      {keyboard.KeyShift, keyboard.KeySemicolon},
		ebiten.KeyNumpadSubtract: {keyboard.KeyHyphen},
		ebiten.KeySemicolon:      {keyboard.KeySemicolon},
		ebiten.KeyQuote:          {keyboard.KeyShift, keyboard.Key7},
		ebiten.KeyComma:          {keyboard.KeyComma},
		ebiten.KeyPeriod:         {keyboard.KeyPeriod},
		ebiten.KeySlash:          {keyboard.KeySlash},
		ebiten.KeyBackslash:      {keyboard.KeyBackslash},
		ebiten.KeyBracketLeft:    {keyboard.KeyLeftBrace},
		ebiten.KeyBracketRight:   {keyboard.KeyRightBrace},
		ebiten.KeyBackquote:      {keyboard.KeyShift, keyboard.KeyBackslash},
		ebiten.KeyF1:             {keyboard.KeyF1},
		ebiten.KeyF2:             {keyboard.KeyF2},
		ebiten.KeyF3:             {keyboard.KeyF3},
		ebiten.KeyF4:             {keyboard.KeyF4},
		ebiten.KeyF5:             {keyboard.KeyF5},
		ebiten.KeyF6:             {keyboard.KeyF6},
		ebiten.KeyF7:             {keyboard.KeyF7},
		ebiten.KeyF8:             {keyboard.KeyF8},
		ebiten.KeyF9:             {keyboard.KeyF9},
		ebiten.KeyF10:            {keyboard.KeyF0},
	}
}

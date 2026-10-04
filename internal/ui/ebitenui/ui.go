//go:build liveui

// Package ebitenui provides the live Ebiten frontend.
package ebitenui

import (
	"image/png"
	"os"
	"sync"
	"time"

	"cpcgo/internal/cpc"
	"cpcgo/internal/keyboard"
	"cpcgo/internal/video"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = video.Width
	screenHeight = video.Height
	sampleRate   = 44100
)

// Config configures the live UI.
type Config struct {
	Scale          int
	ScreenshotPath string
	Mute           bool
}

// Run starts the Ebiten live emulator UI.
func Run(machine *cpc.Machine, config Config) error {
	if config.Scale < 1 {
		config.Scale = 1
	}

	ebiten.SetWindowTitle("cpcgo")
	ebiten.SetWindowSize(screenWidth*config.Scale/2*2, screenHeight*config.Scale/2*2)
	ebiten.SetTPS(50)

	g := &game{
		machine:        machine,
		keys:           defaultKeyMap(),
		screenshotPath: config.ScreenshotPath,
		frame:          ebiten.NewImage(screenWidth, screenHeight),
	}
	if !config.Mute {
		g.startAudio()
	}
	return ebiten.RunGame(g)
}

type game struct {
	machine *cpc.Machine
	keys    map[ebiten.Key]keyboard.Chord
	frame   *ebiten.Image
	audio   *sampleBuffer
	player  *audio.Player

	screenshotPath string
}

func (g *game) startAudio() {
	g.audio = &sampleBuffer{}
	g.machine.SetAudioSink(sampleRate, g.audio.push)
	player, err := audio.NewContext(sampleRate).NewPlayer(g.audio)
	if err != nil {
		return
	}
	player.SetBufferSize(100 * time.Millisecond)
	player.Play()
	g.player = player
}

func (g *game) Update() error {
	g.updateKeyboard()
	g.machine.RunFrame()
	img := g.machine.Monitor().Image()
	g.frame.WritePixels(img.Pix)
	if g.screenshotPath != "" && inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		return saveSnapshot(g.machine, g.screenshotPath)
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.frame, nil)
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
	return png.Encode(file, machine.Monitor().Image())
}

// sampleBuffer queues 16-bit stereo PCM between the emulator and the audio
// player. Reads never block: missing samples play as silence.
type sampleBuffer struct {
	mu   sync.Mutex
	data []byte
}

func (b *sampleBuffer) push(left, right float32) {
	l, r := int16(left*12000), int16(right*12000)
	b.mu.Lock()
	if len(b.data) < sampleRate { // cap latency at about a quarter second
		b.data = append(b.data, byte(l), byte(l>>8), byte(r), byte(r>>8))
	}
	b.mu.Unlock()
}

func (b *sampleBuffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	n := copy(p, b.data)
	b.data = b.data[n:]
	b.mu.Unlock()
	for i := n; i < len(p); i++ {
		p[i] = 0
	}
	return len(p), nil
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

// Package video turns the Gate Array's output into frames.
//
// Monitor models a PAL monitor driven by the Gate Array: a new scanline
// starts on each HSYNC pulse and a new field on each VSYNC pulse; without
// sync pulses the scan free-runs. Each microsecond of a scanline holds 16
// mode 2 pixels.
//
// Frames are captured in the canonical cpc-validation screen format: a window
// 48 microseconds wide and 268 scanlines tall, each scanline doubled, giving a
// 768x536 image. For the firmware's default screen the 640x400 bitmap sits
// 64 pixels from the left edge, centred vertically.
package video

import (
	"image"
	"image/color"

	"cpcgo/internal/gatearray"
)

const (
	// WindowMicros and WindowLines give the captured window size.
	WindowMicros = 48
	WindowLines  = 268

	// Width and Height are the canonical screen.png dimensions.
	Width  = WindowMicros * gatearray.PixelsPerTick
	Height = WindowLines * 2

	// The window starts this many microseconds after the monitor HSYNC and
	// this many scanlines after the monitor VSYNC.
	DefaultOffsetX = 12
	DefaultOffsetY = 34

	// Free-running limits when sync pulses are missing.
	maxLineMicros = 72
	maxFieldLines = 340
)

// Monitor accumulates the Gate Array's output into frames.
type Monitor struct {
	x, y int

	current  [WindowLines][Width]uint8
	complete [WindowLines][Width]uint8
	frames   uint64

	OffsetX, OffsetY int
}

// NewMonitor creates a monitor with the default window position.
func NewMonitor() *Monitor {
	m := &Monitor{OffsetX: DefaultOffsetX, OffsetY: DefaultOffsetY}
	for y := range m.current {
		for x := range m.current[y] {
			m.current[y][x] = gatearray.Black
			m.complete[y][x] = gatearray.Black
		}
	}
	return m
}

// Pixels implements gatearray.Display.
func (m *Monitor) Pixels(px *[gatearray.PixelsPerTick]uint8) {
	col := m.x - m.OffsetX
	line := m.y - m.OffsetY
	if col >= 0 && col < WindowMicros && line >= 0 && line < WindowLines {
		copy(m.current[line][col*gatearray.PixelsPerTick:], px[:])
	}
	m.x++
	if m.x >= maxLineMicros {
		m.HSync()
	}
}

// HSync implements gatearray.Display.
func (m *Monitor) HSync() {
	m.x = 0
	m.y++
	if m.y >= maxFieldLines {
		m.VSync()
	}
}

// VSync implements gatearray.Display.
func (m *Monitor) VSync() {
	m.y = 0
	m.complete = m.current
	m.frames++
}

// Frames returns the number of completed fields.
func (m *Monitor) Frames() uint64 { return m.frames }

// FrameIndices returns the last completed frame as hardware colour numbers,
// one row per scanline (not doubled).
func (m *Monitor) FrameIndices() *[WindowLines][Width]uint8 { return &m.complete }

// Image returns the last completed frame in the canonical format.
func (m *Monitor) Image() *image.RGBA {
	return m.render(&m.complete)
}

// CurrentImage returns the frame being drawn, including any partial field.
func (m *Monitor) CurrentImage() *image.RGBA {
	return m.render(&m.current)
}

func (m *Monitor) render(frame *[WindowLines][Width]uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	for y := 0; y < WindowLines; y++ {
		row := img.Pix[y*2*img.Stride : y*2*img.Stride+Width*4]
		for x, hw := range frame[y] {
			c := HardwareColor(hw)
			row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = c.R, c.G, c.B, 0xff
		}
		copy(img.Pix[(y*2+1)*img.Stride:], row)
	}
	return img
}

// HardwareColor converts a 5-bit Gate Array hardware colour number to RGB,
// using the conventional 0%/50%/100% levels.
func HardwareColor(ink uint8) color.RGBA {
	return hardwarePalette[ink&0x1f]
}

var hardwarePalette = [32]color.RGBA{
	{0x80, 0x80, 0x80, 0xff}, {0x80, 0x80, 0x80, 0xff}, {0x00, 0xff, 0x80, 0xff}, {0xff, 0xff, 0x80, 0xff},
	{0x00, 0x00, 0x80, 0xff}, {0xff, 0x00, 0x80, 0xff}, {0x00, 0x80, 0x80, 0xff}, {0xff, 0x80, 0x80, 0xff},
	{0xff, 0x00, 0x80, 0xff}, {0xff, 0xff, 0x80, 0xff}, {0xff, 0xff, 0x00, 0xff}, {0xff, 0xff, 0xff, 0xff},
	{0xff, 0x00, 0x00, 0xff}, {0xff, 0x00, 0xff, 0xff}, {0xff, 0x80, 0x00, 0xff}, {0xff, 0x80, 0xff, 0xff},
	{0x00, 0x00, 0x80, 0xff}, {0x00, 0xff, 0x80, 0xff}, {0x00, 0xff, 0x00, 0xff}, {0x00, 0xff, 0xff, 0xff},
	{0x00, 0x00, 0x00, 0xff}, {0x00, 0x00, 0xff, 0xff}, {0x00, 0x80, 0x00, 0xff}, {0x00, 0x80, 0xff, 0xff},
	{0x80, 0x00, 0x80, 0xff}, {0x80, 0xff, 0x80, 0xff}, {0x80, 0xff, 0x00, 0xff}, {0x80, 0xff, 0xff, 0xff},
	{0x80, 0x00, 0x00, 0xff}, {0x80, 0x00, 0xff, 0xff}, {0x80, 0x80, 0x00, 0xff}, {0x80, 0x80, 0xff, 0xff},
}

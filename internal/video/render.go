// Package video renders CPC display memory into framebuffers.
package video

import (
	"image"
	"image/color"

	"cpcgo/internal/bus"
	"cpcgo/internal/crtc"
	"cpcgo/internal/gatearray"
)

const (
	Width        = 640
	Height       = 200
	bytesPerLine = 80
)

// Render renders the current CPC display memory to an RGBA image.
func Render(memory *bus.Memory, c *crtc.CRTC, g *gatearray.GateArray) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	mode := g.Mode()
	base := screenBase(c)

	for y := 0; y < Height; y++ {
		lineAddr := base + uint16((y&0x07)*0x0800+(y>>3)*bytesPerLine)
		for byteX := 0; byteX < bytesPerLine; byteX++ {
			val := memory.ReadRAM(lineAddr + uint16(byteX))
			x := byteX * 8
			switch mode {
			case 0:
				plotMode0(img, x, y, val, g)
			case 2:
				plotMode2(img, x, y, val, g)
			default:
				plotMode1(img, x, y, val, g)
			}
		}
	}

	return img
}

func screenBase(c *crtc.CRTC) uint16 {
	r12 := uint16(c.Register(12))
	r13 := uint16(c.Register(13))
	return ((r12 & 0x30) << 10) | (r13 << 1)
}

func plotMode0(img *image.RGBA, x int, y int, val uint8, g *gatearray.GateArray) {
	left := ((val >> 7) & 0x01) << 3
	left |= ((val >> 3) & 0x01) << 2
	left |= ((val >> 5) & 0x01) << 1
	left |= (val >> 1) & 0x01

	right := ((val >> 6) & 0x01) << 3
	right |= ((val >> 2) & 0x01) << 2
	right |= ((val >> 4) & 0x01) << 1
	right |= val & 0x01

	fillRun(img, x, y, 4, inkColor(g, left))
	fillRun(img, x+4, y, 4, inkColor(g, right))
}

func plotMode1(img *image.RGBA, x int, y int, val uint8, g *gatearray.GateArray) {
	pixels := [4]uint8{
		((val>>7)&0x01)<<1 | ((val >> 3) & 0x01),
		((val>>6)&0x01)<<1 | ((val >> 2) & 0x01),
		((val>>5)&0x01)<<1 | ((val >> 1) & 0x01),
		((val>>4)&0x01)<<1 | (val & 0x01),
	}
	for i, pen := range pixels {
		fillRun(img, x+i*2, y, 2, inkColor(g, pen))
	}
}

func plotMode2(img *image.RGBA, x int, y int, val uint8, g *gatearray.GateArray) {
	for i := 0; i < 8; i++ {
		pen := (val >> (7 - i)) & 0x01
		img.SetRGBA(x+i, y, inkColor(g, pen))
	}
}

func fillRun(img *image.RGBA, x int, y int, width int, c color.RGBA) {
	for dx := 0; dx < width; dx++ {
		img.SetRGBA(x+dx, y, c)
	}
}

func inkColor(g *gatearray.GateArray, pen uint8) color.RGBA {
	return hardwareColor(g.Ink(pen))
}

func hardwareColor(ink uint8) color.RGBA {
	if int(ink) >= len(hardwarePalette) {
		return hardwarePalette[0]
	}
	return hardwarePalette[ink]
}

var hardwarePalette = [...]color.RGBA{
	{R: 0x80, G: 0x80, B: 0x80, A: 0xff}, // 0 white, duplicated as grey-ish fallback.
	{R: 0x80, G: 0x80, B: 0x80, A: 0xff},
	{R: 0x00, G: 0xff, B: 0x80, A: 0xff},
	{R: 0xff, G: 0xff, B: 0x80, A: 0xff},
	{R: 0x00, G: 0x00, B: 0x80, A: 0xff},
	{R: 0xff, G: 0x00, B: 0x80, A: 0xff},
	{R: 0x00, G: 0x80, B: 0x80, A: 0xff},
	{R: 0xff, G: 0x80, B: 0x80, A: 0xff},
	{R: 0xff, G: 0x00, B: 0x80, A: 0xff},
	{R: 0xff, G: 0xff, B: 0x80, A: 0xff},
	{R: 0xff, G: 0xff, B: 0x00, A: 0xff},
	{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	{R: 0xff, G: 0x00, B: 0x00, A: 0xff},
	{R: 0xff, G: 0x00, B: 0xff, A: 0xff},
	{R: 0xff, G: 0x80, B: 0x00, A: 0xff},
	{R: 0xff, G: 0x80, B: 0xff, A: 0xff},
	{R: 0x00, G: 0x00, B: 0x80, A: 0xff},
	{R: 0x00, G: 0xff, B: 0x80, A: 0xff},
	{R: 0x00, G: 0xff, B: 0x00, A: 0xff},
	{R: 0x00, G: 0xff, B: 0xff, A: 0xff},
	{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
	{R: 0x00, G: 0x00, B: 0xff, A: 0xff},
	{R: 0x00, G: 0x80, B: 0x00, A: 0xff},
	{R: 0x00, G: 0x80, B: 0xff, A: 0xff},
	{R: 0x80, G: 0x00, B: 0x80, A: 0xff},
	{R: 0x80, G: 0xff, B: 0x80, A: 0xff},
	{R: 0x80, G: 0xff, B: 0x00, A: 0xff},
	{R: 0x80, G: 0xff, B: 0xff, A: 0xff},
	{R: 0x80, G: 0x00, B: 0x00, A: 0xff},
	{R: 0x80, G: 0x00, B: 0xff, A: 0xff},
	{R: 0x80, G: 0x80, B: 0x00, A: 0xff},
	{R: 0x80, G: 0x80, B: 0xff, A: 0xff},
}

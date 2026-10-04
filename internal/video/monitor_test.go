package video

import (
	"testing"

	"cpcgo/internal/gatearray"
)

func TestMonitorWindowAndVerticalHold(t *testing.T) {
	m := NewMonitor()
	var px [gatearray.PixelsPerTick]uint8
	for i := range px {
		px[i] = 11
	}
	// One field: VSYNC at line 0 (and optionally again mid-field), then
	// lines of 64 microseconds starting at HSYNC.
	field := func(extraVSync int) {
		for line := 0; line < 312; line++ {
			if line == 0 || line == extraVSync {
				m.VSync()
			}
			m.HSync()
			for x := 0; x < 64; x++ {
				m.Pixels(&px)
			}
		}
	}
	field(-1)
	field(-1)
	frames := m.Frames()
	field(150) // the mid-field VSYNC is ignored
	if m.Frames() != frames+1 {
		t.Fatalf("one field with a mid-field VSYNC counted %d frames, want 1", m.Frames()-frames)
	}
	if got := m.Image().Bounds(); got.Dx() != Width || got.Dy() != Height {
		t.Fatalf("image size %v", got)
	}
}

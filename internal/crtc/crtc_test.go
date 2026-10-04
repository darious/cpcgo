package crtc

import "testing"

// frameStats runs one frame worth of characters from the start of a frame
// and records where the sync and display signals change.
type frameStats struct {
	charsPerLine, lines       int
	hsyncStart, hsyncWidth    int
	vsyncLine, vsyncLines     int
	displayLines, displayCols int
}

func measure(c *CRTC) frameStats {
	var s frameStats
	// Run until the start of a frame.
	for !(c.VCC() == 0 && c.VLC() == 0 && c.HCC() == 0) {
		c.Tick()
	}
	line, col := 0, 0
	s.vsyncLine = -1
	prevV := false
	for {
		if c.HSync() && line == 0 {
			if s.hsyncWidth == 0 {
				s.hsyncStart = col
			}
			s.hsyncWidth++
		}
		if c.DisplayEnabled() && line == 0 {
			s.displayCols++
		}
		if col == 0 {
			if c.VSync() && !prevV {
				s.vsyncLine = line
			}
			if c.VSync() {
				s.vsyncLines++
			}
			prevV = c.VSync()
			if c.vdisp {
				s.displayLines++
			}
		}
		c.Tick()
		col++
		if c.HCC() == 0 {
			if line == 0 {
				s.charsPerLine = col
			}
			col = 0
			line++
			if c.VCC() == 0 && c.VLC() == 0 {
				s.lines = line
				return s
			}
		}
	}
}

func TestStandardFrameTiming(t *testing.T) {
	for _, tc := range []struct {
		typ        int
		vsyncLines int
	}{{0, 8}, {1, 16}, {2, 16}, {4, 8}} {
		s := measure(NewType(tc.typ))
		want := frameStats{
			charsPerLine: 64, lines: 312, hsyncStart: 46, hsyncWidth: 14,
			vsyncLine: 240, vsyncLines: tc.vsyncLines, displayLines: 200, displayCols: 40,
		}
		if s != want {
			t.Errorf("type %d: %+v, want %+v", tc.typ, s, want)
		}
	}
}

func TestVerticalAdjustAndRowLength(t *testing.T) {
	c := NewType(1)
	c.SetRegister(9, 3)
	c.SetRegister(4, 76)
	c.SetRegister(5, 4)
	c.SetRegister(6, 50)
	c.SetRegister(7, 60)
	measure(c) // let the new values take effect from a frame start
	s := measure(c)
	if s.lines != 77*4+4 || s.displayLines != 200 || s.vsyncLine != 240 {
		t.Fatalf("got %+v, want 312 lines, 200 displayed, VSYNC at 240", s)
	}
}

func TestR7WriteMatchingRowStartsVSync(t *testing.T) {
	c := NewType(1)
	for c.VCC() != 5 {
		c.Tick()
	}
	for i := 0; i < 100; i++ {
		c.Tick()
	}
	c.WritePort(0xbc00, 7)
	c.WritePort(0xbd00, c.VCC())
	if !c.VSync() {
		t.Fatal("writing R7 = VCC did not start VSYNC")
	}
}

func TestType0ShowsFirstLineWithR6Zero(t *testing.T) {
	for _, typ := range []int{0, 1} {
		c := NewType(typ)
		c.SetRegister(6, 0)
		measure(c)
		s := measure(c)
		want := 0
		if typ == 0 {
			want = 1
		}
		if s.displayLines != want {
			t.Errorf("type %d with R6=0: %d displayed lines, want %d", typ, s.displayLines, want)
		}
	}
}

func TestRegisterReadsByType(t *testing.T) {
	for _, tc := range []struct {
		typ          int
		r12, r14, r3 uint8
		status       uint8
	}{
		{0, 0x12, 0x14, 0, 0xff},
		{1, 0, 0x14, 0, 0},
		{2, 0, 0x14, 0, 0xff},
		{4, 0x12, 0x14, 0x13, 0x13},
	} {
		c := NewType(tc.typ)
		c.SetRegister(11, 0x13)
		c.SetRegister(12, 0x12)
		c.SetRegister(14, 0x14)
		read := func(r uint8, port uint16) uint8 {
			c.WritePort(0xbc00, r)
			v, _ := c.ReadPort(port)
			return v
		}
		got := []uint8{read(12, 0xbf00), read(14, 0xbf00), read(3, 0xbf00), read(3, 0xbe00)}
		want := []uint8{tc.r12, tc.r14, tc.r3, tc.status}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("type %d: reads %#v, want %#v", tc.typ, got, want)
				break
			}
		}
	}
}

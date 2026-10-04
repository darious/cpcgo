package fdc

import (
	"testing"

	"cpcgo/internal/dsk"
)

func testDisk() *dsk.Disk {
	d := &dsk.Disk{Sides: 1}
	for t := 0; t < 3; t++ {
		tr := &dsk.Track{Cylinder: uint8(t)}
		for r := 0; r < 9; r++ {
			data := make([]byte, 512)
			data[0] = uint8(t*16 + r)
			tr.Sectors = append(tr.Sectors, &dsk.Sector{C: uint8(t), R: uint8(0xc1 + r), N: 2, Data: data})
		}
		d.SetTrack(t, 0, tr)
	}
	return d
}

func send(t *testing.T, f *FDC, bytes ...uint8) {
	t.Helper()
	for _, b := range bytes {
		if s, _ := f.ReadPort(0xfb7e); s&msrRQM == 0 || s&msrDIO != 0 {
			t.Fatalf("FDC not ready for a command byte (MSR %#02x)", s)
		}
		f.WritePort(0xfb7f, b)
	}
}

// readAll collects execution-phase data and result bytes.
func readAll(f *FDC) (data, result []uint8) {
	for i := 0; i < 100000; i++ {
		s, _ := f.ReadPort(0xfb7e)
		if s&msrRQM == 0 {
			f.Tick()
			continue
		}
		if s&msrDIO == 0 {
			return
		}
		v, _ := f.ReadPort(0xfb7f)
		if s&msrEXM != 0 {
			data = append(data, v)
		} else {
			result = append(result, v)
		}
	}
	return
}

func TestReadyChangeInterruptAndSeek(t *testing.T) {
	f := New()
	f.Insert(0, testDisk())
	f.WritePort(0xfa7e, 1) // motor on: drive A becomes ready
	send(t, f, 0x08)
	if _, res := readAll(f); len(res) != 2 || res[0] != 0xc0 {
		t.Fatalf("sense interrupt after motor on = %x, want c0 00", res)
	}
	send(t, f, 0x0f, 0x00, 0x02) // seek to track 2
	send(t, f, 0x08)
	if _, res := readAll(f); len(res) != 2 || res[0] != 0x20 || res[1] != 2 {
		t.Fatalf("sense interrupt after seek = %x, want 20 02", res)
	}
	send(t, f, 0x08)
	if _, res := readAll(f); len(res) != 1 || res[0] != 0x80 {
		t.Fatalf("sense interrupt with nothing pending = %x, want 80", res)
	}
}

func TestReadDataToEndOfCylinder(t *testing.T) {
	f := New()
	f.Insert(0, testDisk())
	f.WritePort(0xfa7e, 1)
	send(t, f, 0x0f, 0x00, 0x01)
	send(t, f, 0x46, 0x00, 0x01, 0x00, 0xc3, 0x02, 0xc4, 0x2a, 0xff)
	data, res := readAll(f)
	if len(data) != 1024 || data[0] != 0x12 || data[512] != 0x13 {
		t.Fatalf("read %d bytes, first bytes %x %x", len(data), data[0], data[512])
	}
	want := []uint8{0x40, 0x80, 0x00, 0x01, 0x00, 0xc4, 0x02}
	for i := range want {
		if res[i] != want[i] {
			t.Fatalf("result = %x, want %x", res, want)
		}
	}
}

func TestFMReadIDFindsNoAddressMark(t *testing.T) {
	f := New()
	f.Insert(0, testDisk())
	f.WritePort(0xfa7e, 1)
	send(t, f, 0x0a, 0x00)
	if _, res := readAll(f); res[0] != 0x40 || res[1] != 0x01 {
		t.Fatalf("FM READ ID = %x, want 40 01 ...", res)
	}
	send(t, f, 0x4a, 0x00)
	if _, res := readAll(f); res[0] != 0x00 || res[5] != 0xc1 {
		t.Fatalf("MFM READ ID = %x, want first sector C1", res)
	}
}

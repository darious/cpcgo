package dsk

import (
	"bytes"
	"testing"
)

func standardImage() []byte {
	header := make([]byte, 0x100)
	copy(header, "MV - CPCEMU Disk-File\r\nDisk-Info\r\n")
	header[0x30], header[0x31] = 2, 1
	header[0x32], header[0x33] = 0x00, 0x13 // 0x1300 bytes per track
	out := header
	for t := 0; t < 2; t++ {
		info := make([]byte, 0x100)
		copy(info, "Track-Info\r\n")
		info[0x10], info[0x14], info[0x15] = byte(t), 2, 9
		for i := 0; i < 9; i++ {
			copy(info[0x18+i*8:], []byte{byte(t), 0, byte(0xc1 + i), 2})
		}
		data := bytes.Repeat([]byte{0xe5}, 9*512)
		data[0] = byte(t + 1)
		out = append(out, append(info, data...)...)
	}
	return out
}

func TestParseStandardAndRoundTrip(t *testing.T) {
	d, err := Parse(standardImage())
	if err != nil {
		t.Fatal(err)
	}
	tr := d.Track(1, 0)
	if d.Tracks != 2 || tr == nil || len(tr.Sectors) != 9 || tr.Sectors[0].R != 0xc1 || tr.Sectors[0].Data[0] != 2 {
		t.Fatalf("parsed disk = %+v, track 1 = %+v", d, tr)
	}
	again, err := Parse(d.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if !again.Extended || again.Track(1, 0).Sectors[8].R != 0xc9 || len(again.Track(0, 0).Sectors[3].Data) != 512 {
		t.Fatal("extended round trip lost data")
	}
}

func TestRejectsUnknownFormat(t *testing.T) {
	if _, err := Parse(make([]byte, 512)); err == nil {
		t.Fatal("accepted a non-DSK image")
	}
}

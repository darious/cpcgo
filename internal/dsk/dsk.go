// Package dsk reads and writes CPCEMU disk images: the standard "MV - CPC"
// format and the "EXTENDED CPC DSK" format.
//
// Layout: a 256-byte disk header, then one block per track (track 0 side 0,
// track 0 side 1, ...). Each track block starts with a 256-byte "Track-Info"
// header holding up to 29 eight-byte sector descriptors (C, H, R, N, ST1,
// ST2 and, in extended images, the stored data length) followed by the
// sector data in descriptor order.
package dsk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

const (
	standardSignature = "MV - CPC"
	extendedSignature = "EXTENDED CPC DSK File"
	trackSignature    = "Track-Info"
	headerSize        = 0x100
)

// Sector is one sector on a track.
type Sector struct {
	C, H, R, N uint8
	ST1, ST2   uint8
	Data       []byte
}

// Track is the list of sectors on one side of one cylinder, in the order
// they pass under the head.
type Track struct {
	Cylinder, Side uint8
	Gap3, Filler   uint8
	Sectors        []*Sector
}

// Disk is an in-memory disk image.
type Disk struct {
	Creator  string
	Tracks   int
	Sides    int
	Extended bool
	// Track returns nil for unformatted tracks.
	tracks [][]*Track
	Dirty  bool
}

// Load reads a disk image file.
func Load(path string) (*Disk, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return d, nil
}

// Parse decodes a disk image.
func Parse(data []byte) (*Disk, error) {
	if len(data) < headerSize {
		return nil, errors.New("disk image too short")
	}
	d := &Disk{}
	switch {
	case bytes.HasPrefix(data, []byte(extendedSignature)):
		d.Extended = true
	case bytes.HasPrefix(data, []byte(standardSignature)):
	default:
		return nil, errors.New("not a CPCEMU disk image")
	}
	d.Creator = string(bytes.TrimRight(data[0x22:0x30], "\x00 "))
	d.Tracks = int(data[0x30])
	d.Sides = int(data[0x31])
	if d.Sides < 1 || d.Sides > 2 {
		return nil, fmt.Errorf("unsupported side count %d", d.Sides)
	}
	d.tracks = make([][]*Track, d.Tracks)
	for i := range d.tracks {
		d.tracks[i] = make([]*Track, d.Sides)
	}

	offset := headerSize
	standardSize := int(binary.LittleEndian.Uint16(data[0x32:0x34]))
	for t := 0; t < d.Tracks; t++ {
		for s := 0; s < d.Sides; s++ {
			size := standardSize
			if d.Extended {
				size = int(data[0x34+t*d.Sides+s]) * 256
			}
			if size == 0 {
				continue
			}
			if offset+size > len(data) {
				return nil, fmt.Errorf("track %d side %d extends past end of image", t, s)
			}
			track, err := parseTrack(data[offset:offset+size], d.Extended)
			if err != nil {
				return nil, fmt.Errorf("track %d side %d: %w", t, s, err)
			}
			d.tracks[t][s] = track
			offset += size
		}
	}
	return d, nil
}

func parseTrack(block []byte, extended bool) (*Track, error) {
	if len(block) < headerSize || !bytes.HasPrefix(block, []byte(trackSignature)) {
		return nil, errors.New("missing Track-Info header")
	}
	t := &Track{
		Cylinder: block[0x10],
		Side:     block[0x11],
		Gap3:     block[0x16],
		Filler:   block[0x17],
	}
	sectorSize := sectorLength(block[0x14])
	count := int(block[0x15])
	if count > 29 {
		return nil, fmt.Errorf("too many sectors (%d)", count)
	}
	offset := headerSize
	for i := 0; i < count; i++ {
		info := block[0x18+i*8 : 0x20+i*8]
		s := &Sector{C: info[0], H: info[1], R: info[2], N: info[3], ST1: info[4], ST2: info[5]}
		length := sectorSize
		if extended {
			length = int(binary.LittleEndian.Uint16(info[6:8]))
		} else if s.N < 8 {
			length = sectorLength(s.N)
			if length > sectorSize {
				length = sectorSize
			}
		}
		if offset+length > len(block) {
			length = len(block) - offset
			if length < 0 {
				length = 0
			}
		}
		s.Data = append([]byte(nil), block[offset:offset+length]...)
		offset += length
		t.Sectors = append(t.Sectors, s)
	}
	return t, nil
}

func sectorLength(n uint8) int {
	if n > 6 {
		n = 6
	}
	return 128 << n
}

// Track returns the given track, or nil if it is absent or unformatted.
func (d *Disk) Track(cylinder, side int) *Track {
	if cylinder < 0 || cylinder >= len(d.tracks) || side < 0 || side >= d.Sides {
		return nil
	}
	return d.tracks[cylinder][side]
}

// SetTrack replaces a track (used by FORMAT TRACK), growing the image when
// needed.
func (d *Disk) SetTrack(cylinder, side int, t *Track) {
	for cylinder >= len(d.tracks) {
		d.tracks = append(d.tracks, make([]*Track, d.Sides))
	}
	if side >= d.Sides {
		return
	}
	if cylinder >= d.Tracks {
		d.Tracks = cylinder + 1
	}
	d.tracks[cylinder][side] = t
	d.Dirty = true
}

// Encode serialises the disk as an extended image.
func (d *Disk) Encode() []byte {
	var out bytes.Buffer
	header := make([]byte, headerSize)
	copy(header, extendedSignature+"\r\nDisk-Info\r\n")
	copy(header[0x22:], "cpcgo")
	header[0x30] = uint8(d.Tracks)
	header[0x31] = uint8(d.Sides)
	var blocks [][]byte
	for t := 0; t < d.Tracks; t++ {
		for s := 0; s < d.Sides; s++ {
			var block []byte
			if tr := d.Track(t, s); tr != nil {
				block = encodeTrack(tr)
			}
			header[0x34+t*d.Sides+s] = uint8(len(block) / 256)
			blocks = append(blocks, block)
		}
	}
	out.Write(header)
	for _, b := range blocks {
		out.Write(b)
	}
	return out.Bytes()
}

func encodeTrack(t *Track) []byte {
	header := make([]byte, headerSize)
	copy(header, trackSignature+"\r\n")
	header[0x10] = t.Cylinder
	header[0x11] = t.Side
	if len(t.Sectors) > 0 {
		header[0x14] = t.Sectors[0].N
	}
	header[0x15] = uint8(len(t.Sectors))
	header[0x16] = t.Gap3
	header[0x17] = t.Filler
	var data []byte
	for i, s := range t.Sectors {
		info := header[0x18+i*8:]
		info[0], info[1], info[2], info[3], info[4], info[5] = s.C, s.H, s.R, s.N, s.ST1, s.ST2
		binary.LittleEndian.PutUint16(info[6:], uint16(len(s.Data)))
		data = append(data, s.Data...)
	}
	block := append(header, data...)
	if rem := len(block) % 256; rem != 0 {
		block = append(block, make([]byte, 256-rem)...)
	}
	return block
}

// Save writes the disk to path as an extended image.
func (d *Disk) Save(path string) error {
	if err := os.WriteFile(path, d.Encode(), 0o644); err != nil {
		return err
	}
	d.Dirty = false
	return nil
}

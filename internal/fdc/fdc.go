// Package fdc emulates the NEC uPD765A floppy disk controller as wired in the
// CPC 664/6128 and the DDI-1 interface.
//
// Ports: &FA7E (write) drive motor; &FB7E (read) main status register;
// &FB7F data register. Only the US0 drive-select line is connected, so drives
// 0/2 are A: and 1/3 are B:. There is no DMA, no interrupt line and no
// terminal count, so multi-sector transfers end at EOT with the "end of
// cylinder" status that AMSDOS expects.
//
// Each command goes through the datasheet's three phases: command bytes in,
// execution (data transfer, one byte every 32 microseconds), result bytes
// out.
package fdc

import "cpcgo/internal/dsk"

// Main status register bits.
const (
	msrRQM = 0x80 // request for master: data register ready
	msrDIO = 0x40 // data direction: FDC to CPU
	msrEXM = 0x20 // execution phase
	msrCB  = 0x10 // controller busy
)

// Status register 0-3 bits used here.
const (
	st0AbnormalTermination = 0x40
	st0InvalidCommand      = 0x80
	st0SeekEnd             = 0x20
	st0NotReady            = 0x08

	st1EndOfCylinder  = 0x80
	st1DataError      = 0x20
	st1NoData         = 0x04
	st1NotWritable    = 0x02
	st1MissingAddress = 0x01

	st2ControlMark = 0x40
	st2DataError   = 0x20

	st3WriteProtect = 0x40
	st3Ready        = 0x20
	st3Track0       = 0x10
	st3TwoSide      = 0x08
)

// byteMicros is the time between data bytes at 250 kbit/s MFM.
const byteMicros = 32

type phase int

const (
	phaseIdle phase = iota
	phaseCommand
	phaseExecRead
	phaseExecWrite
	phaseResult
)

// Drive is a disk drive attached to the controller.
type Drive struct {
	Disk         *dsk.Disk
	WriteProtect bool
	track        int
	sectorPos    int // index of the sector next under the head
	seekPending  bool
	seekStatus   uint8
	ready        bool
	doubleSided  bool
}

// FDC is the controller state.
type FDC struct {
	Drives [2]Drive
	motor  bool

	phase   phase
	command []byte
	want    int
	result  []byte

	// params holds command bytes 2-5. The uPD765 keeps them in the
	// registers it reports as C, H, R, N when a command fails without
	// reading an ID.
	params [4]uint8

	// Execution phase state.
	data      []byte
	dataPos   int
	readyAt   uint64
	clock     uint64
	transfer  transferState
	lastDrive int
}

type transferState struct {
	write      bool
	drive      int
	head       int
	c, h, r, n uint8
	eot        uint8
	dtl        uint8
	skip       bool
	deleted    bool // read/write deleted data command
	sector     *dsk.Sector
	st0        uint8
	st1, st2   uint8
	format     bool
	formatN    uint8
	formatSC   int
	formatGap  uint8
	formatFill uint8
	readTrack  bool
	trackIndex int
}

// New creates an idle controller with two empty drives.
func New() *FDC {
	f := &FDC{}
	f.Drives[0].doubleSided = false
	f.Drives[1].doubleSided = true
	return f
}

// Insert puts a disk into drive 0 (A:) or 1 (B:).
func (f *FDC) Insert(drive int, d *dsk.Disk) {
	f.Drives[drive&1].Disk = d
	f.Drives[drive&1].sectorPos = 0
	f.updateReady()
}

// updateReady raises the interrupt the uPD765 generates when a drive's READY
// line changes (ST0 = abnormal termination by ready change + drive).
func (f *FDC) updateReady() {
	for i := range f.Drives {
		d := &f.Drives[i]
		ready := f.ready(d)
		if ready != d.ready {
			d.ready = ready
			d.seekPending = true
			d.seekStatus = 0xc0 | uint8(i)
		}
	}
}

// Motor reports whether the drive motor is on.
func (f *FDC) Motor() bool { return f.motor }

// Tick advances the controller clock by one microsecond.
func (f *FDC) Tick() { f.clock++ }

// ReadPort implements bus.IODevice. The FDC is decoded when A10 and A7 are
// low and A8 is high; A0 selects the status (0) or data (1) register.
func (f *FDC) ReadPort(port uint16) (uint8, bool) {
	if port&0x0580 != 0x0100 {
		return 0, false
	}
	if port&1 == 0 {
		return f.status(), true
	}
	return f.readData(), true
}

// WritePort implements bus.IODevice.
func (f *FDC) WritePort(port uint16, val uint8) bool {
	if port&0x0480 != 0 {
		return false
	}
	if port&0x0100 == 0 {
		f.motor = val&1 != 0
		f.updateReady()
		return true
	}
	if port&1 == 1 {
		f.writeData(val)
	}
	return true
}

func (f *FDC) status() uint8 {
	var busy uint8
	for i := range f.Drives {
		if f.Drives[i].seekPending {
			busy |= 1 << i
		}
	}
	switch f.phase {
	case phaseIdle:
		return msrRQM | busy
	case phaseCommand:
		return msrRQM | msrCB | busy
	case phaseExecRead:
		if f.clock >= f.readyAt {
			return msrRQM | msrDIO | msrEXM | msrCB
		}
		return msrDIO | msrEXM | msrCB
	case phaseExecWrite:
		if f.clock >= f.readyAt {
			return msrRQM | msrEXM | msrCB
		}
		return msrEXM | msrCB
	case phaseResult:
		return msrRQM | msrDIO | msrCB
	}
	return msrRQM
}

func (f *FDC) readData() uint8 {
	switch f.phase {
	case phaseExecRead:
		v := f.data[f.dataPos]
		f.dataPos++
		f.readyAt = f.clock + byteMicros
		if f.dataPos >= len(f.data) {
			f.nextReadSector()
		}
		return v
	case phaseResult:
		v := f.result[0]
		f.result = f.result[1:]
		if len(f.result) == 0 {
			f.phase = phaseIdle
		}
		return v
	}
	return 0xff
}

func (f *FDC) writeData(val uint8) {
	switch f.phase {
	case phaseIdle:
		f.command = append(f.command[:0], val)
		f.want = commandLength(val)
		f.phase = phaseCommand
		if f.want == 1 {
			f.execute()
		}
	case phaseCommand:
		f.command = append(f.command, val)
		if len(f.command) >= f.want {
			f.execute()
		}
	case phaseExecWrite:
		f.data[f.dataPos] = val
		f.dataPos++
		f.readyAt = f.clock + byteMicros
		if f.dataPos >= len(f.data) {
			f.finishWriteBlock()
		}
	}
}

// commandLength returns the number of command bytes, including the first.
func commandLength(cmd uint8) int {
	switch cmd & 0x1f {
	case 0x02, 0x05, 0x06, 0x09, 0x0c, 0x11, 0x19, 0x1d:
		return 9
	case 0x03, 0x0f:
		return 3
	case 0x04, 0x07, 0x0a:
		return 2
	case 0x0d:
		return 6
	}
	return 1
}

func (f *FDC) setResult(bytes ...uint8) {
	f.result = append(f.result[:0], bytes...)
	f.phase = phaseResult
}

func (f *FDC) drive(us uint8) *Drive { return &f.Drives[us&1] }

func (f *FDC) ready(d *Drive) bool { return d.Disk != nil && f.motor }

func (f *FDC) execute() {
	cmd := f.command
	if len(cmd) >= 6 {
		copy(f.params[:], cmd[2:6])
	}
	switch cmd[0] & 0x1f {
	case 0x03: // SPECIFY
		f.phase = phaseIdle
	case 0x04: // SENSE DRIVE STATUS
		us := cmd[1] & 0x07
		d := f.drive(us)
		st3 := us & 0x07
		if f.ready(d) {
			st3 |= st3Ready
		}
		if d.track == 0 {
			st3 |= st3Track0
		}
		if d.doubleSided {
			st3 |= st3TwoSide
		}
		if d.WriteProtect || d.Disk == nil {
			st3 |= st3WriteProtect
		}
		f.setResult(st3)
	case 0x07: // RECALIBRATE
		f.seek(cmd[1]&0x03, 0)
	case 0x0f: // SEEK
		f.seek(cmd[1]&0x07, int(cmd[2]))
	case 0x08: // SENSE INTERRUPT STATUS
		for i := range f.Drives {
			d := &f.Drives[i]
			if d.seekPending {
				d.seekPending = false
				f.setResult(d.seekStatus, uint8(d.track))
				return
			}
		}
		f.setResult(st0InvalidCommand)
	case 0x0a: // READ ID
		f.readID(cmd[0], cmd[1])
	case 0x06, 0x0c: // READ DATA, READ DELETED DATA
		f.startTransfer(false)
	case 0x05, 0x09: // WRITE DATA, WRITE DELETED DATA
		f.startTransfer(true)
	case 0x02: // READ TRACK
		f.startReadTrack()
	case 0x0d: // FORMAT TRACK
		f.startFormat()
	default: // includes SCAN commands, which AMSDOS never uses
		f.setResult(st0InvalidCommand)
	}
}

func (f *FDC) seek(us uint8, track int) {
	d := f.drive(us)
	d.track = track
	d.seekPending = true
	d.seekStatus = st0SeekEnd | us&0x07
	if d.Disk == nil {
		d.seekStatus |= st0NotReady
	}
	f.phase = phaseIdle
}

func (f *FDC) currentTrack(d *Drive, head int) *dsk.Track {
	if d.Disk == nil {
		return nil
	}
	if head >= d.Disk.Sides {
		return nil
	}
	return d.Disk.Track(d.track, head)
}

func (f *FDC) readID(cmd, hdus uint8) {
	us := hdus & 0x07
	head := int(hdus>>2) & 1
	d := f.drive(us)
	st0 := hdus & 0x07
	if !f.ready(d) {
		f.setResult(st0|st0AbnormalTermination|st0NotReady, 0, 0, 0, 0, 0, 0)
		return
	}
	t := f.currentTrack(d, head)
	// Disk images hold MFM tracks: an FM (MF=0) search finds no ID.
	if t == nil || len(t.Sectors) == 0 || cmd&0x40 == 0 {
		p := f.params
		f.setResult(st0|st0AbnormalTermination, st1MissingAddress, 0, p[0], p[1], p[2], p[3])
		return
	}
	d.sectorPos %= len(t.Sectors)
	s := t.Sectors[d.sectorPos]
	d.sectorPos = (d.sectorPos + 1) % len(t.Sectors)
	f.setResult(st0, 0, 0, s.C, s.H, s.R, s.N)
}

// findSector searches the track for the sector with the requested ID,
// starting at the current rotational position, as the FDC does over two
// revolutions.
func (f *FDC) findSector(d *Drive, t *dsk.Track, c, h, r, n uint8) *dsk.Sector {
	count := len(t.Sectors)
	for i := 0; i < count; i++ {
		idx := (d.sectorPos + i) % count
		s := t.Sectors[idx]
		if s.C == c && s.H == h && s.R == r && s.N == n {
			d.sectorPos = (idx + 1) % count
			return s
		}
	}
	return nil
}

func (f *FDC) startTransfer(write bool) {
	cmd := f.command
	tr := transferState{
		write:   write,
		drive:   int(cmd[1] & 1),
		head:    int(cmd[1]>>2) & 1,
		c:       cmd[2],
		h:       cmd[3],
		r:       cmd[4],
		n:       cmd[5],
		eot:     cmd[6],
		dtl:     cmd[8],
		skip:    cmd[0]&0x20 != 0,
		deleted: cmd[0]&0x1f == 0x0c || cmd[0]&0x1f == 0x09,
		st0:     cmd[1] & 0x07,
	}
	f.transfer = tr
	f.lastDrive = tr.drive
	d := f.drive(cmd[1])
	if !f.ready(d) {
		f.endTransfer(st0AbnormalTermination|st0NotReady, 0, 0)
		return
	}
	if cmd[0]&0x40 == 0 {
		f.endTransfer(st0AbnormalTermination, st1MissingAddress, 0)
		return
	}
	if write && (d.WriteProtect) {
		f.endTransfer(st0AbnormalTermination, st1NotWritable, 0)
		return
	}
	f.beginSector()
}

func (f *FDC) sectorSize() int {
	tr := &f.transfer
	if tr.n == 0 {
		return int(tr.dtl)
	}
	n := tr.n
	if n > 6 {
		n = 6
	}
	return 128 << n
}

// beginSector locates sector R and starts transferring it.
func (f *FDC) beginSector() {
	tr := &f.transfer
	d := &f.Drives[tr.drive]
	t := f.currentTrack(d, tr.head)
	if t == nil || len(t.Sectors) == 0 {
		f.endTransfer(st0AbnormalTermination, st1MissingAddress, 0)
		return
	}
	s := f.findSector(d, t, tr.c, tr.h, tr.r, tr.n)
	if s == nil {
		f.endTransfer(st0AbnormalTermination, st1NoData, 0)
		return
	}
	tr.sector = s
	size := f.sectorSize()
	if tr.write {
		f.data = make([]byte, size)
		f.phase = phaseExecWrite
	} else {
		isDeleted := s.ST2&st2ControlMark != 0
		if isDeleted != tr.deleted {
			tr.st2 |= st2ControlMark
			if tr.skip {
				f.advanceSector()
				return
			}
		}
		f.data = make([]byte, size)
		copy(f.data, s.Data)
		for i := len(s.Data); i < size; i++ {
			f.data[i] = 0xe5
		}
		f.phase = phaseExecRead
	}
	f.dataPos = 0
	f.readyAt = f.clock + byteMicros
}

func (f *FDC) nextReadSector() {
	tr := &f.transfer
	s := tr.sector
	if s.ST1&st1DataError != 0 || s.ST2&st2DataError != 0 {
		f.endTransfer(st0AbnormalTermination, s.ST1&st1DataError, s.ST2&st2DataError|tr.st2)
		return
	}
	if tr.st2&st2ControlMark != 0 {
		// A sector of the other data type ends the transfer after it.
		f.endTransferNext(0, 0)
		return
	}
	f.advanceSector()
}

func (f *FDC) finishWriteBlock() {
	tr := &f.transfer
	if tr.format {
		f.finishFormat()
		return
	}
	s := tr.sector
	s.Data = append(s.Data[:0], f.data...)
	if tr.deleted {
		s.ST2 |= st2ControlMark
	} else {
		s.ST2 &^= st2ControlMark
	}
	f.Drives[tr.drive].Disk.Dirty = true
	f.advanceSector()
}

// advanceSector moves to the next sector, ending the command after EOT.
func (f *FDC) advanceSector() {
	tr := &f.transfer
	if tr.r == tr.eot {
		f.endTransferNext(st0AbnormalTermination, st1EndOfCylinder)
		return
	}
	tr.r++
	f.beginSector()
}

// endTransferNext ends a transfer after the sector just completed. Without
// a terminal count the result reports the ID of that last sector.
func (f *FDC) endTransferNext(st0, st1 uint8) {
	f.endTransfer(st0, st1, f.transfer.st2)
}

func (f *FDC) endTransfer(st0, st1, st2 uint8) {
	tr := &f.transfer
	f.setResult(tr.st0|uint8(tr.head<<2)|st0, tr.st1|st1, st2, tr.c, tr.h, tr.r, tr.n)
}

func (f *FDC) startReadTrack() {
	cmd := f.command
	f.transfer = transferState{
		drive: int(cmd[1] & 1), head: int(cmd[1]>>2) & 1,
		c: cmd[2], h: cmd[3], r: cmd[4], n: cmd[5], eot: cmd[6], dtl: cmd[8],
		st0: cmd[1] & 0x07, readTrack: true,
	}
	d := f.drive(cmd[1])
	if !f.ready(d) {
		f.endTransfer(st0AbnormalTermination|st0NotReady, 0, 0)
		return
	}
	t := f.currentTrack(d, f.transfer.head)
	if t == nil || len(t.Sectors) == 0 {
		f.endTransfer(st0AbnormalTermination, st1MissingAddress, 0)
		return
	}
	// Concatenate sectors from the index hole up to EOT sectors.
	size := f.sectorSize()
	var data []byte
	for i := 0; i < len(t.Sectors) && i < int(f.transfer.eot); i++ {
		s := t.Sectors[i]
		buf := make([]byte, size)
		copy(buf, s.Data)
		data = append(data, buf...)
		f.transfer.r = s.R
	}
	f.transfer.sector = &dsk.Sector{}
	f.transfer.eot = f.transfer.r
	f.data = data
	f.dataPos = 0
	f.phase = phaseExecRead
	f.readyAt = f.clock + byteMicros
}

func (f *FDC) startFormat() {
	cmd := f.command
	f.transfer = transferState{
		write: true, format: true,
		drive: int(cmd[1] & 1), head: int(cmd[1]>>2) & 1,
		formatN: cmd[2], formatSC: int(cmd[3]), formatGap: cmd[4], formatFill: cmd[5],
		st0: cmd[1] & 0x07, n: cmd[2],
	}
	d := f.drive(cmd[1])
	if !f.ready(d) {
		f.endTransfer(st0AbnormalTermination|st0NotReady, 0, 0)
		return
	}
	if d.WriteProtect {
		f.endTransfer(st0AbnormalTermination, st1NotWritable, 0)
		return
	}
	f.data = make([]byte, 4*f.transfer.formatSC)
	f.dataPos = 0
	f.phase = phaseExecWrite
	f.readyAt = f.clock + byteMicros
}

func (f *FDC) finishFormat() {
	tr := &f.transfer
	d := &f.Drives[tr.drive]
	n := tr.formatN
	if n > 6 {
		n = 6
	}
	t := &dsk.Track{Cylinder: uint8(d.track), Side: uint8(tr.head), Gap3: tr.formatGap, Filler: tr.formatFill}
	for i := 0; i < tr.formatSC; i++ {
		id := f.data[i*4 : i*4+4]
		data := make([]byte, 128<<n)
		for j := range data {
			data[j] = tr.formatFill
		}
		t.Sectors = append(t.Sectors, &dsk.Sector{C: id[0], H: id[1], R: id[2], N: id[3], Data: data})
		tr.c, tr.h, tr.r = id[0], id[1], id[2]
	}
	d.Disk.SetTrack(d.track, tr.head, t)
	// With no terminal count the format runs on to the index hole and
	// ends with "end of cylinder"; C, H, R, N hold the command parameters.
	p := f.params
	f.setResult(tr.st0|uint8(tr.head<<2)|st0AbnormalTermination, st1EndOfCylinder, 0, p[0], p[1], p[2], p[3])
}

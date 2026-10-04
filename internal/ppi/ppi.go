// Package ppi emulates the Intel 8255 PPI as wired in the CPC.
//
// Port A: PSG data bus (input or output). Port B (input): bit 0 VSYNC,
// bits 1-3 distributor ID, bit 4 screen refresh (1 = 50 Hz), bit 5 /EXP,
// bit 6 printer BUSY, bit 7 cassette read data. Port C (output): bits 0-3
// keyboard line, bit 4 cassette motor, bit 5 cassette write, bits 6-7 PSG
// BC1/BDIR.
package ppi

import "cpcgo/internal/psg"

// Port B inputs other than VSYNC and /EXP: Amstrad distributor ID (7),
// 50 Hz, printer BUSY high (no printer), no cassette signal.
const defaultPortB = 0x0e | 0x10 | 0x40

// portBExp is the /EXP input: high unless an expansion pulls it low. The
// 664/6128's built-in disk interface (and a DDI-1 on the 464) does.
const portBExp = 0x20

// PPI holds the 8255 port latches and control word.
type PPI struct {
	portA   uint8
	portC   uint8
	control uint8
	psg     *psg.PSG

	// VSync reports the CRTC VSYNC signal for port B bit 0.
	VSync func() bool
	// PortBInputs overrides bits 1-7 of port B when non-zero.
	PortBInputs uint8
	// Expansion pulls /EXP (port B bit 5) low.
	Expansion bool
}

// New creates a PPI connected to the PSG.
func New(psgDevice *psg.PSG) *PPI {
	return &PPI{control: 0x9b, psg: psgDevice}
}

// ReadPort implements bus.IODevice. The PPI is selected when A11 is low.
func (p *PPI) ReadPort(port uint16) (uint8, bool) {
	if port&0x0800 != 0 {
		return 0, false
	}
	switch (port >> 8) & 0x03 {
	case 0:
		if p.control&0x10 != 0 { // port A input
			return p.psgBus(), true
		}
		return p.portA, true
	case 1:
		return p.PortB(), true
	case 2:
		return p.portC, true
	}
	return 0xff, true
}

// WritePort implements bus.IODevice.
func (p *PPI) WritePort(port uint16, val uint8) bool {
	if port&0x0800 != 0 {
		return false
	}
	switch (port >> 8) & 0x03 {
	case 0:
		p.portA = val
		p.drivePSG()
	case 1:
		// Port B is an input on the CPC.
	case 2:
		p.portC = val
		p.drivePSG()
	case 3:
		if val&0x80 != 0 {
			// Mode set: all output latches are cleared.
			p.control = val
			p.portA, p.portC = 0, 0
		} else {
			bit := (val >> 1) & 0x07
			if val&1 != 0 {
				p.portC |= 1 << bit
			} else {
				p.portC &^= 1 << bit
			}
		}
		p.drivePSG()
	}
	return true
}

// PortB returns the port B input value.
func (p *PPI) PortB() uint8 {
	v := uint8(defaultPortB)
	if !p.Expansion {
		v |= portBExp
	}
	if p.PortBInputs != 0 {
		v = p.PortBInputs &^ 1
	}
	if p.VSync != nil && p.VSync() {
		v |= 0x01
	}
	return v
}

// PortA returns the port A output latch.
func (p *PPI) PortA() uint8 { return p.portA }

// PortC returns the port C output latch.
func (p *PPI) PortC() uint8 { return p.portC }

// Control returns the last mode-set control word.
func (p *PPI) Control() uint8 { return p.control }

// KeyboardLine returns the keyboard line selected by port C.
func (p *PPI) KeyboardLine() uint8 { return p.portC & 0x0f }

// MotorOn reports the cassette motor bit.
func (p *PPI) MotorOn() bool { return p.portC&0x10 != 0 }

// drivePSG performs the PSG bus operation selected by port C bits 6-7
// (00 inactive, 01 read, 10 write, 11 latch address). Writes only reach the
// PSG when port A is an output.
func (p *PPI) drivePSG() {
	if p.psg == nil || p.control&0x10 != 0 {
		return
	}
	switch p.portC & 0xc0 {
	case 0x80:
		p.psg.Write(p.portA)
	case 0xc0:
		p.psg.Select(p.portA)
	}
}

// psgBus returns what the PSG drives onto port A.
func (p *PPI) psgBus() uint8 {
	if p.psg == nil {
		return 0xff
	}
	if p.portC&0xc0 == 0x40 {
		return p.psg.Read()
	}
	return 0xff
}

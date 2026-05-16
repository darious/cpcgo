// Package ppi emulates the Intel 8255 PPI.
package ppi

// PPI stores the CPC-visible 8255 ports and control state.
type PPI struct {
	portA   uint8
	portB   uint8
	portC   uint8
	control uint8
}

// New creates a PPI with CPC-like reset inputs.
func New() *PPI {
	return &PPI{
		portB:   0xfe,
		control: 0x9b,
	}
}

// ReadPort implements bus.IODevice.
func (p *PPI) ReadPort(port uint16) (uint8, bool) {
	if !selectedByPort(port) {
		return 0, false
	}

	switch registerSelect(port) {
	case 0:
		return p.portA, true
	case 1:
		return p.portB, true
	case 2:
		return p.portC, true
	default:
		return 0xff, true
	}
}

// WritePort implements bus.IODevice.
func (p *PPI) WritePort(port uint16, val uint8) bool {
	if !selectedByPort(port) {
		return false
	}

	switch registerSelect(port) {
	case 0:
		p.portA = val
	case 1:
		p.portB = val
	case 2:
		p.portC = val
	case 3:
		p.writeControl(val)
	}
	return true
}

// PortA returns the latched port A value.
func (p *PPI) PortA() uint8 {
	return p.portA
}

// PortB returns the latched port B value.
func (p *PPI) PortB() uint8 {
	return p.portB
}

// PortC returns the latched port C value.
func (p *PPI) PortC() uint8 {
	return p.portC
}

// Control returns the last mode-set control value.
func (p *PPI) Control() uint8 {
	return p.control
}

func (p *PPI) writeControl(val uint8) {
	if val&0x80 != 0 {
		p.control = val
		return
	}

	bit := (val >> 1) & 0x07
	if val&0x01 != 0 {
		p.portC |= 1 << bit
		return
	}
	p.portC &^= 1 << bit
}

func selectedByPort(port uint16) bool {
	return port&0x0800 == 0
}

func registerSelect(port uint16) uint8 {
	return uint8((port >> 8) & 0x03)
}

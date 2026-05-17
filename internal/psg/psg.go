// Package psg emulates the AY-3-8912 sound generator.
package psg

const RegisterCount = 16

// PSG stores AY-3-8912 register state.
type PSG struct {
	selected  uint8
	registers [RegisterCount]uint8
}

// New creates a PSG.
func New() *PSG {
	return &PSG{}
}

// Select selects an AY register.
func (p *PSG) Select(register uint8) {
	p.selected = register & 0x0f
}

// Selected returns the selected AY register.
func (p *PSG) Selected() uint8 {
	return p.selected
}

// Write writes to the selected AY register.
func (p *PSG) Write(val uint8) {
	p.registers[p.selected] = val
}

// Read reads from the selected AY register.
func (p *PSG) Read() uint8 {
	return p.registers[p.selected]
}

// Register returns an AY register value.
func (p *PSG) Register(register uint8) uint8 {
	if register >= RegisterCount {
		return 0
	}
	return p.registers[register]
}

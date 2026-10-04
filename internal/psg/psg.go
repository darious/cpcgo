// Package psg emulates the General Instrument AY-3-8912 sound generator.
//
// On the CPC the PSG is clocked at 1 MHz and is only reachable through the
// 8255 PPI: PPI port A carries the data bus and port C bits 6-7 drive BDIR and
// BC1. The chip's single I/O port (register 14) reads the keyboard matrix.
package psg

const (
	RegisterCount = 16

	// ClockHz is the PSG clock on the CPC.
	ClockHz = 1_000_000
	// TickHz is the rate at which Tick must be called (clock / 8).
	TickHz = ClockHz / 8
)

// Read-back masks: unused register bits read as zero.
var registerMasks = [RegisterCount]uint8{
	0xff, 0x0f, 0xff, 0x0f, 0xff, 0x0f, 0x1f, 0xff,
	0x1f, 0x1f, 0x1f, 0xff, 0xff, 0x0f, 0xff, 0xff,
}

// Logarithmic output levels for amplitudes 0-15 (about 3 dB per step),
// normalised to 1.0.
var volumeTable = [16]float32{
	0, 0.0106, 0.0150, 0.0222, 0.0320, 0.0466, 0.0665, 0.1039,
	0.1237, 0.2020, 0.2484, 0.3550, 0.4677, 0.6052, 0.7743, 1.0,
}

// PSG holds AY-3-8912 register and sound generator state.
type PSG struct {
	selected  uint8
	registers [RegisterCount]uint8

	// PortA supplies the value read from I/O port A (register 14) when the
	// port is in input mode. On the CPC it returns the selected keyboard row.
	PortA func() uint8

	toneCount  [3]uint16
	toneOutput [3]bool
	noiseCount uint16
	noiseLFSR  uint32
	noiseOut   bool
	envCount   uint32
	envStep    int
	envHolding bool
	envAttack  bool
	subTick    bool
}

// New creates a PSG in its reset state.
func New() *PSG {
	p := &PSG{noiseLFSR: 1}
	p.registers[14] = 0xff
	p.registers[15] = 0xff
	return p
}

// Select latches a register address. Addresses 16-255 select nothing.
func (p *PSG) Select(register uint8) {
	p.selected = register
}

// Selected returns the latched register address.
func (p *PSG) Selected() uint8 { return p.selected }

// Write writes to the selected register.
func (p *PSG) Write(val uint8) {
	p.SetRegister(p.selected, val)
}

// Read reads from the selected register.
func (p *PSG) Read() uint8 {
	if p.selected >= RegisterCount {
		return 0xff
	}
	if p.selected == 14 && p.registers[7]&0x40 == 0 {
		if p.PortA != nil {
			return p.PortA()
		}
		return 0xff
	}
	return p.registers[p.selected] & registerMasks[p.selected]
}

// Register returns a register value as last written (masked).
func (p *PSG) Register(register uint8) uint8 {
	if register >= RegisterCount {
		return 0
	}
	return p.registers[register] & registerMasks[register]
}

// SetRegister sets a register value directly.
func (p *PSG) SetRegister(register uint8, val uint8) {
	if register >= RegisterCount {
		return
	}
	p.registers[register] = val
	if register == 13 {
		p.restartEnvelope()
	}
}

// ---------------------------------------------------------------------------
// Sound generation

func (p *PSG) tonePeriod(ch int) uint16 {
	period := uint16(p.registers[ch*2]) | uint16(p.registers[ch*2+1]&0x0f)<<8
	if period == 0 {
		period = 1
	}
	return period
}

func (p *PSG) restartEnvelope() {
	p.envCount = 0
	p.envStep = 0
	p.envHolding = false
	p.envAttack = p.registers[13]&0x04 != 0
}

// envelopeLevel returns the current envelope amplitude (0-15).
func (p *PSG) envelopeLevel() int {
	if p.envAttack {
		return p.envStep
	}
	return 15 - p.envStep
}

func (p *PSG) stepEnvelope() {
	if p.envHolding {
		return
	}
	p.envStep++
	if p.envStep < 16 {
		return
	}
	shape := p.registers[13] & 0x0f
	cont, attack, alt, hold := shape&8 != 0, shape&4 != 0, shape&2 != 0, shape&1 != 0
	if !cont {
		// Shapes 0-7: one ramp, then hold at zero.
		p.envHolding = true
		p.envStep = 15
		p.envAttack = false
		return
	}
	if hold {
		p.envHolding = true
		p.envStep = 15
		if alt {
			p.envAttack = !attack
		} else {
			p.envAttack = attack
		}
		return
	}
	p.envStep = 0
	if alt {
		p.envAttack = !p.envAttack
	}
}

// Tick advances the generators by 8 PSG clocks (one 125 kHz step).
func (p *PSG) Tick() {
	for ch := 0; ch < 3; ch++ {
		p.toneCount[ch]++
		if p.toneCount[ch] >= p.tonePeriod(ch) {
			p.toneCount[ch] = 0
			p.toneOutput[ch] = !p.toneOutput[ch]
		}
	}

	// The noise generator and envelope run at half the tone rate.
	p.subTick = !p.subTick
	if p.subTick {
		noisePeriod := uint16(p.registers[6] & 0x1f)
		if noisePeriod == 0 {
			noisePeriod = 1
		}
		p.noiseCount++
		if p.noiseCount >= noisePeriod {
			p.noiseCount = 0
			// 17-bit LFSR with taps at bits 0 and 3.
			bit := (p.noiseLFSR ^ (p.noiseLFSR >> 3)) & 1
			p.noiseLFSR = p.noiseLFSR>>1 | bit<<16
			p.noiseOut = p.noiseLFSR&1 != 0
		}
	}

	// One envelope step takes 16*EP clocks (two ticks per period unit),
	// so a full 16-step ramp lasts 256*EP clocks.
	if p.subTick {
		envPeriod := uint32(p.registers[11]) | uint32(p.registers[12])<<8
		if envPeriod == 0 {
			envPeriod = 1
		}
		p.envCount++
		if p.envCount >= envPeriod {
			p.envCount = 0
			p.stepEnvelope()
		}
	}
}

// Channel returns the instantaneous output level (0..1) of channel 0-2.
func (p *PSG) Channel(ch int) float32 {
	mixer := p.registers[7]
	toneOff := mixer&(1<<ch) != 0
	noiseOff := mixer&(8<<ch) != 0
	on := (toneOff || p.toneOutput[ch]) && (noiseOff || p.noiseOut)
	if !on {
		return 0
	}
	amp := p.registers[8+ch] & 0x1f
	if amp&0x10 != 0 {
		return volumeTable[p.envelopeLevel()]
	}
	return volumeTable[amp&0x0f]
}

// Stereo returns the CPC's ABC stereo mix: A left, B centre, C right.
func (p *PSG) Stereo() (left, right float32) {
	a, b, c := p.Channel(0), p.Channel(1), p.Channel(2)
	return (a + b*0.5) / 1.5, (c + b*0.5) / 1.5
}

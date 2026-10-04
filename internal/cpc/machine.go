// Package cpc coordinates the emulated CPC machine.
//
// Timing: the Z80 runs with CPC wait states, so its T-state counter divided
// by four is the machine time in microseconds. The video hardware (CRTC, Gate
// Array, monitor) and the PSG are advanced one microsecond at a time to catch
// up with the CPU before every memory write and I/O access, and after every
// instruction, so device state is exact at each access to within the
// microsecond.
package cpc

import (
	"fmt"
	"image"

	"cpcgo/internal/bus"
	"cpcgo/internal/crtc"
	"cpcgo/internal/fdc"
	"cpcgo/internal/gatearray"
	"cpcgo/internal/keyboard"
	"cpcgo/internal/ppi"
	"cpcgo/internal/psg"
	"cpcgo/internal/rom"
	"cpcgo/internal/video"
	"cpcgo/internal/z80"
)

// Model identifies the CPC model being emulated.
type Model string

const (
	Model464  Model = "464"
	Model664  Model = "664"
	Model6128 Model = "6128"
)

// Config contains startup configuration for a CPC machine.
type Config struct {
	Model    Model
	ROMs     rom.Image
	CRTCType int
	// DiskInterface fits a floppy disk controller (always on 664/6128;
	// on a 464 it models a DDI-1 interface). AMSDOS must be in ROMs.
	DiskInterface bool
	Disk          string
	Scale         int
}

// Machine is the top-level emulator state.
type Machine struct {
	config Config
	memory *bus.Memory
	io     *bus.IO
	cpu    *z80.CPU

	gateArray *gatearray.GateArray
	romSelect *gatearray.ROMSelect
	crtc      *crtc.CRTC
	ppi       *ppi.PPI
	psg       *psg.PSG
	keyboard  *keyboard.Matrix
	monitor   *video.Monitor
	fdc       *fdc.FDC

	micros     uint64
	psgPhase   uint8
	audioSink  func(left, right float32)
	sampleStep float64
	samplePos  float64
}

// New constructs a machine from validated configuration.
func New(config Config) (*Machine, error) {
	if config.Model == "" {
		config.Model = Model6128
	}
	banks := 4
	switch config.Model {
	case Model464:
	case Model664:
		config.DiskInterface = true
	case Model6128:
		config.DiskInterface = true
		banks = 8
	default:
		return nil, fmt.Errorf("unsupported model %q", config.Model)
	}
	if config.CRTCType < 0 || config.CRTCType > 4 {
		return nil, fmt.Errorf("unsupported CRTC type %d", config.CRTCType)
	}
	if config.DiskInterface && len(config.ROMs.AMSDOS) == 0 {
		config.DiskInterface = false
	}
	roms := config.ROMs
	if !config.DiskInterface {
		roms.AMSDOS = nil
	}

	memory, err := bus.NewMemoryWithBanks(roms, banks)
	if err != nil {
		return nil, err
	}

	m := &Machine{
		config:    config,
		memory:    memory,
		io:        bus.NewIO(),
		gateArray: gatearray.New(memory),
		romSelect: gatearray.NewROMSelect(memory),
		crtc:      crtc.NewType(config.CRTCType),
		psg:       psg.New(),
		keyboard:  keyboard.New(),
		monitor:   video.NewMonitor(),
	}
	if config.CRTCType >= 3 {
		m.gateArray.InkDelay = 1
	}
	m.ppi = ppi.New(m.psg)
	m.ppi.VSync = m.crtc.VSync
	m.psg.PortA = func() uint8 { return m.keyboard.Row(m.ppi.KeyboardLine()) }

	m.io.Add(m.gateArray)
	m.io.Add(m.romSelect)
	m.io.Add(m.crtc)
	m.io.Add(m.ppi)
	if config.DiskInterface {
		m.ppi.Expansion = true
		m.fdc = fdc.New()
		m.io.Add(m.fdc)
	}

	m.cpu = z80.New(&cpuBus{m: m})
	m.cpu.WaitStates = true
	return m, nil
}

// cpuBus connects the Z80 to memory and I/O, synchronising the rest of the
// machine before every access that can affect it.
type cpuBus struct{ m *Machine }

func (b *cpuBus) Read(addr uint16) uint8 { return b.m.memory.Read(addr) }

func (b *cpuBus) Write(addr uint16, val uint8) {
	b.m.sync()
	b.m.memory.Write(addr, val)
}

func (b *cpuBus) In(port uint16) uint8 {
	b.m.sync()
	return b.m.io.In(port)
}

func (b *cpuBus) Out(port uint16, val uint8) {
	if port&0x4000 == 0 && b.m.crtc.Type() < 3 {
		// The discrete CRTCs (types 0-2) latch a write one microsecond
		// earlier in the OUT cycle than the integrated type 3/4 parts.
		b.m.syncTo(b.m.cpu.Cycles()/4 - 1)
	} else {
		b.m.sync()
	}
	b.m.io.Out(port, val)
}

// IntAck implements z80.InterruptAcknowledger. The CPC data bus floats high
// during the acknowledge, which IM 2 software relies on.
func (b *cpuBus) IntAck() uint8 {
	b.m.sync()
	b.m.gateArray.Acknowledge()
	b.m.cpu.INT(false, 0xff)
	return 0xff
}

// sync advances the devices to the CPU's current time.
func (m *Machine) sync() {
	m.syncTo(m.cpu.Cycles() / 4)
}

// syncTo advances the devices to the given microsecond.
func (m *Machine) syncTo(target uint64) {
	for m.micros < target {
		m.tick()
	}
}

// tick advances the video hardware, PSG and FDC by one microsecond.
func (m *Machine) tick() {
	m.gateArray.Clock(m.crtc, m.monitor)
	m.crtc.Tick()
	m.micros++

	m.psgPhase++
	if m.psgPhase == 8 {
		m.psgPhase = 0
		m.psg.Tick()
		if m.audioSink != nil {
			m.samplePos += m.sampleStep
			for m.samplePos >= 1 {
				m.samplePos--
				m.audioSink(m.psg.Stereo())
			}
		}
	}
	if m.fdc != nil {
		m.fdc.Tick()
	}
}

// SetAudioSink installs a callback that receives stereo samples at rate Hz.
func (m *Machine) SetAudioSink(rate int, sink func(left, right float32)) {
	m.audioSink = sink
	if rate > 0 {
		m.sampleStep = float64(rate) / psg.TickHz
	}
}

// Step executes one CPU instruction or interrupt response.
func (m *Machine) Step() int {
	cycles := m.cpu.Step()
	m.sync()
	m.cpu.INT(m.gateArray.IRQ(), 0xff)
	return cycles
}

// RunInstructions executes count CPU steps and returns consumed T-states.
func (m *Machine) RunInstructions(count int) uint64 {
	var cycles uint64
	for i := 0; i < count; i++ {
		cycles += uint64(m.Step())
	}
	return cycles
}

// RunFrame runs until the monitor completes a frame (a VSYNC, or the
// monitor's free-running limit when the software produces no VSYNC).
func (m *Machine) RunFrame() uint64 {
	start := m.monitor.Frames()
	startCycles := m.cpu.Cycles()
	for m.monitor.Frames() == start {
		m.Step()
	}
	return m.cpu.Cycles() - startCycles
}

// RunMicros runs for at least n microseconds of machine time.
func (m *Machine) RunMicros(n uint64) {
	end := m.micros + n
	for m.micros < end {
		m.Step()
	}
}

// Config returns the machine startup configuration.
func (m *Machine) Config() Config { return m.config }

// Memory returns the machine memory map.
func (m *Machine) Memory() *bus.Memory { return m.memory }

// IO returns the machine I/O dispatcher.
func (m *Machine) IO() *bus.IO { return m.io }

// CPU returns the Z80 CPU.
func (m *Machine) CPU() *z80.CPU { return m.cpu }

// GateArray returns the machine Gate Array.
func (m *Machine) GateArray() *gatearray.GateArray { return m.gateArray }

// CRTC returns the machine CRTC.
func (m *Machine) CRTC() *crtc.CRTC { return m.crtc }

// PPI returns the machine PPI.
func (m *Machine) PPI() *ppi.PPI { return m.ppi }

// PSG returns the machine PSG.
func (m *Machine) PSG() *psg.PSG { return m.psg }

// FDC returns the floppy disk controller, or nil without a disk interface.
func (m *Machine) FDC() *fdc.FDC { return m.fdc }

// Keyboard returns the machine keyboard matrix.
func (m *Machine) Keyboard() *keyboard.Matrix { return m.keyboard }

// Monitor returns the display.
func (m *Machine) Monitor() *video.Monitor { return m.monitor }

// Micros returns the machine time in microseconds.
func (m *Machine) Micros() uint64 { return m.micros }

// BorderInk returns the current Gate Array hardware colour for the border.
func (m *Machine) BorderInk() uint8 { return m.gateArray.Ink(gatearray.BorderPen) }

// Framebuffer returns the last completed frame in the canonical format.
func (m *Machine) Framebuffer() image.Image { return m.monitor.Image() }

// Reset resets the CPU. Memory and devices keep their current state.
func (m *Machine) Reset() {
	m.cpu.Reset()
	m.cpu.INT(false, 0xff)
}

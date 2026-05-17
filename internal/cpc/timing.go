package cpc

const (
	cyclesPerSecond     = 4_000_000
	framesPerSecond     = 50
	interruptsPerSecond = 300

	cyclesPerFrame     = cyclesPerSecond / framesPerSecond
	cyclesPerInterrupt = cyclesPerSecond / interruptsPerSecond
	vsyncCycles        = cyclesPerFrame / 40
)

// FramesPerSecond is the PAL CPC frame cadence used by the current scheduler.
const FramesPerSecond = framesPerSecond

// TimingStats reports coarse machine timing counters.
type TimingStats struct {
	Cycles     uint64
	Frames     uint64
	Interrupts uint64
	FrameCycle uint64
	VSync      bool
}

type timingState struct {
	cycles         uint64
	frames         uint64
	interrupts     uint64
	frameCycle     uint64
	interruptCycle uint64
	vsync          bool
}

func (t *timingState) reset() {
	*t = timingState{}
}

func (t *timingState) advance(cycles int) bool {
	if cycles <= 0 {
		return false
	}

	n := uint64(cycles)
	t.cycles += n
	t.frameCycle += n
	t.interruptCycle += n

	for t.frameCycle >= cyclesPerFrame {
		t.frameCycle -= cyclesPerFrame
		t.frames++
	}

	interrupt := false
	for t.interruptCycle >= cyclesPerInterrupt {
		t.interruptCycle -= cyclesPerInterrupt
		t.interrupts++
		interrupt = true
	}

	t.vsync = t.frameCycle >= cyclesPerFrame-vsyncCycles
	return interrupt
}

func (t timingState) stats() TimingStats {
	return TimingStats{
		Cycles:     t.cycles,
		Frames:     t.frames,
		Interrupts: t.interrupts,
		FrameCycle: t.frameCycle,
		VSync:      t.vsync,
	}
}

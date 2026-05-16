package bus

// Bus combines the CPC memory map and I/O dispatcher behind the Z80 bus API.
type Bus struct {
	Memory *Memory
	IO     *IO
}

// New creates a CPU bus from memory and I/O devices.
func New(memory *Memory, io *IO) *Bus {
	if io == nil {
		io = NewIO()
	}
	return &Bus{Memory: memory, IO: io}
}

// Fetch reads an opcode byte during M1.
func (b *Bus) Fetch(addr uint16) uint8 {
	return b.Memory.Fetch(addr)
}

// Read reads CPU memory.
func (b *Bus) Read(addr uint16) uint8 {
	return b.Memory.Read(addr)
}

// Write writes CPU memory.
func (b *Bus) Write(addr uint16, val uint8) {
	b.Memory.Write(addr, val)
}

// In reads CPU I/O.
func (b *Bus) In(port uint16) uint8 {
	return b.IO.In(port)
}

// Out writes CPU I/O.
func (b *Bus) Out(port uint16, val uint8) {
	b.IO.Out(port, val)
}

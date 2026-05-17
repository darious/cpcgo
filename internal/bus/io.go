package bus

// IODevice handles CPC I/O port reads and writes.
type IODevice interface {
	ReadPort(port uint16) (uint8, bool)
	WritePort(port uint16, val uint8) bool
}

// IO dispatches CPC I/O port access with explicit partial decoding in devices.
type IO struct {
	devices     []IODevice
	defaultRead uint8
	observer    IOObserver
}

// IOOperation identifies a port operation type.
type IOOperation uint8

const (
	IORead IOOperation = iota
	IOWrite
)

// IOEvent describes a completed I/O access.
type IOEvent struct {
	Operation IOOperation
	Port      uint16
	Value     uint8
	Handled   bool
}

// IOObserver receives I/O events after dispatch.
type IOObserver func(IOEvent)

// NewIO creates an I/O dispatcher.
func NewIO() *IO {
	return &IO{defaultRead: 0xff}
}

// Add registers an I/O device. Earlier devices win read conflicts; writes are
// broadcast to all matching devices.
func (io *IO) Add(device IODevice) {
	io.devices = append(io.devices, device)
}

// SetObserver installs an I/O observer and returns the previous observer.
func (io *IO) SetObserver(observer IOObserver) IOObserver {
	previous := io.observer
	io.observer = observer
	return previous
}

// In reads a byte from an I/O port.
func (io *IO) In(port uint16) uint8 {
	for _, device := range io.devices {
		if val, ok := device.ReadPort(port); ok {
			io.observe(IOEvent{Operation: IORead, Port: port, Value: val, Handled: true})
			return val
		}
	}
	io.observe(IOEvent{Operation: IORead, Port: port, Value: io.defaultRead, Handled: false})
	return io.defaultRead
}

// Out writes a byte to an I/O port.
func (io *IO) Out(port uint16, val uint8) {
	handled := false
	for _, device := range io.devices {
		if device.WritePort(port, val) {
			handled = true
		}
	}
	io.observe(IOEvent{Operation: IOWrite, Port: port, Value: val, Handled: handled})
}

func (io *IO) observe(event IOEvent) {
	if io.observer != nil {
		io.observer(event)
	}
}

// Device is a small function-backed I/O device useful for tests and simple
// hardware adapters.
type Device struct {
	Read  func(port uint16) (uint8, bool)
	Write func(port uint16, val uint8) bool
}

// ReadPort implements IODevice.
func (d Device) ReadPort(port uint16) (uint8, bool) {
	if d.Read == nil {
		return 0, false
	}
	return d.Read(port)
}

// WritePort implements IODevice.
func (d Device) WritePort(port uint16, val uint8) bool {
	if d.Write == nil {
		return false
	}
	return d.Write(port, val)
}

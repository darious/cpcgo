package bus

import "testing"

func TestIOReadFirstMatchAndDefault(t *testing.T) {
	io := NewIO()
	io.Add(Device{Read: func(port uint16) (uint8, bool) {
		return 0x11, port&0xff00 == 0x7f00
	}})
	io.Add(Device{Read: func(port uint16) (uint8, bool) {
		return 0x22, port&0xff00 == 0x7f00
	}})

	if got := io.In(0x7f10); got != 0x11 {
		t.Fatalf("matched read = %#02x, want %#02x", got, 0x11)
	}
	if got := io.In(0xbf10); got != 0xff {
		t.Fatalf("default read = %#02x, want %#02x", got, 0xff)
	}
}

func TestIOWriteBroadcastsToAllMatches(t *testing.T) {
	io := NewIO()
	var first, second []uint16
	match7Fxx := func(port uint16) bool { return port&0xff00 == 0x7f00 }

	io.Add(Device{Write: func(port uint16, val uint8) bool {
		if !match7Fxx(port) {
			return false
		}
		first = append(first, port|uint16(val)<<8)
		return true
	}})
	io.Add(Device{Write: func(port uint16, val uint8) bool {
		if !match7Fxx(port) {
			return false
		}
		second = append(second, port|uint16(val)<<8)
		return true
	}})

	io.Out(0x7f20, 0x55)
	io.Out(0xbf20, 0x66)

	if len(first) != 1 || first[0] != 0x7f20|0x5500 {
		t.Fatalf("first writes = %#v", first)
	}
	if len(second) != 1 || second[0] != 0x7f20|0x5500 {
		t.Fatalf("second writes = %#v", second)
	}
}

func TestIOObserverReportsHandledAndUnhandledAccesses(t *testing.T) {
	io := NewIO()
	var events []IOEvent
	io.SetObserver(func(event IOEvent) {
		events = append(events, event)
	})
	io.Add(Device{
		Read: func(port uint16) (uint8, bool) {
			return 0x77, port == 0x1000
		},
		Write: func(port uint16, val uint8) bool {
			return port == 0x2000 && val == 0x88
		},
	})

	_ = io.In(0x1000)
	_ = io.In(0x1001)
	io.Out(0x2000, 0x88)
	io.Out(0x2001, 0x99)

	want := []IOEvent{
		{Operation: IORead, Port: 0x1000, Value: 0x77, Handled: true},
		{Operation: IORead, Port: 0x1001, Value: 0xff, Handled: false},
		{Operation: IOWrite, Port: 0x2000, Value: 0x88, Handled: true},
		{Operation: IOWrite, Port: 0x2001, Value: 0x99, Handled: false},
	}
	if len(events) != len(want) {
		t.Fatalf("event count = %d, want %d: %#v", len(events), len(want), events)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("event %d = %#v, want %#v", i, events[i], want[i])
		}
	}
}

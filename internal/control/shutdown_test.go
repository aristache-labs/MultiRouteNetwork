package control

import "testing"

type closingSwitch struct {
	testSwitch
	closed bool
}

func (s *closingSwitch) Close() { s.closed = true }
func TestShutdownClosesKeptDevice(t *testing.T) {
	s := New("", nil)
	sw := &closingSwitch{}
	sw.on.Store(true)
	s.sw = sw
	s.Stop()
	if sw.closed {
		t.Fatal("Stop closed the kept device")
	}
	s.Shutdown()
	if !sw.closed || sw.IsOn() {
		t.Fatal("Exit did not close the device")
	}
}

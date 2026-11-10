package control

import (
	"testing"

	"MultiRouteNetwork/internal/engine"
)

type closingSwitch struct {
	testSwitch
	closed bool
}

func (s *closingSwitch) Close() { s.closed = true }
func TestQuitInServiceModeStopsRoutingAndReturns(t *testing.T) {
	EnableStayResident()
	t.Cleanup(func() { stayResident.Store(false) })
	s := New("", nil)
	sw := &testSwitch{}
	sw.On()
	s.Attach(engine.NewScheduler(nil), sw)
	s.Quit()
	if sw.IsOn() {
		t.Fatal("service quit left routing on")
	}
}

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

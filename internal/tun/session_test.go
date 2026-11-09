package tun

import (
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"net"
	"sync"
	"testing"
	"time"
)

func TestStopClosesTCPUDPAndRejectsLateConnections(t *testing.T) {
	s := newFlowSession()
	client, peer := net.Pipe()
	defer peer.Close()
	release, ok := s.track(client)
	if !ok {
		t.Fatal("TCP registration")
	}
	udp, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.track(udp); !ok {
		t.Fatal("UDP registration")
	}
	tcpDone, udpDone := make(chan error, 1), make(chan error, 1)
	go func() { _, err := client.Read(make([]byte, 1)); tcpDone <- err }()
	go func() { _, _, err := udp.ReadFrom(make([]byte, 1)); udpDone <- err }()
	s.stop()
	s.stop()
	release()
	for _, done := range []chan error{tcpDone, udpDone} {
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("read survived Stop")
			}
		case <-time.After(time.Second):
			t.Fatal("Stop did not unblock read")
		}
	}
	select {
	case <-s.ctx.Done():
	default:
		t.Fatal("pending dial context survived Stop")
	}
	late, latePeer := net.Pipe()
	defer latePeer.Close()
	if _, ok := s.track(late); ok {
		t.Fatal("accepted late dial")
	}
	if _, err := late.Write([]byte{1}); err == nil {
		t.Fatal("late connection not closed")
	}
	next := newFlowSession()
	defer next.stop()
	c, p := net.Pipe()
	defer p.Close()
	if _, ok := next.track(c); !ok {
		t.Fatal("new generation rejected")
	}
	s.stop()
	next.mu.Lock()
	count := len(next.flows)
	next.mu.Unlock()
	if count != 1 {
		t.Fatal("old Stop affected new generation")
	}
}

func TestSessionConcurrentRegistrationAndStop(t *testing.T) {
	for i := 0; i < 20; i++ {
		s := newFlowSession()
		var wg sync.WaitGroup
		for j := 0; j < 20; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				a, b := net.Pipe()
				defer b.Close()
				release, ok := s.track(a)
				if ok {
					release()
					a.Close()
				}
			}()
		}
		s.stop()
		wg.Wait()
		s.mu.Lock()
		n := len(s.flows)
		s.mu.Unlock()
		if n != 0 {
			t.Fatal("leaked connection")
		}
	}
}

type lifecycleDevice struct {
	stack.LinkEndpoint
	closes int
}

func (d *lifecycleDevice) Name() string { return "fake" }
func (d *lifecycleDevice) Type() string { return "test" }
func (d *lifecycleDevice) Close()       { d.closes++ }

// All route/device operations are fake: this test never creates a real TUN or changes routes.
func TestFastPauseResumeKeepsDeviceAndCloseReleasesIt(t *testing.T) {
	previous := pauseCapture
	defer func() { pauseCapture = previous }()
	pauses, resumes := 0, 0
	pauseCapture = func(int) (func() error, error) { pauses++; return func() error { resumes++; return nil }, nil }
	d := &lifecycleDevice{}
	tun := &Tun{dev: d}
	tun.on.Store(true)
	tun.flows.Store(newFlowSession())
	for i := 0; i < 10; i++ {
		old := tun.flows.Load()
		tun.Off()
		tun.Off()
		if tun.IsOn() || tun.dev != d || d.closes != 0 {
			t.Fatal("pause destroyed device")
		}
		select {
		case <-old.ctx.Done():
		default:
			t.Fatal("old session still active")
		}
		if err := tun.On(); err != nil {
			t.Fatal(err)
		}
		if err := tun.On(); err != nil {
			t.Fatal(err)
		}
		if !tun.IsOn() || tun.dev != d || tun.flows.Load() == old {
			t.Fatal("resume did not create a fresh session")
		}
	}
	if pauses != 10 || resumes != 10 {
		t.Fatalf("non-idempotent transitions %d/%d", pauses, resumes)
	}
	tun.Off()
	tun.Close()
	tun.Close()
	if tun.dev != nil || tun.stk != nil || tun.flows.Load() != nil || d.closes != 1 {
		t.Fatal("Exit did not fully release device")
	}
}

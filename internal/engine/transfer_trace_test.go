package engine

import (
	"io"
	"net"
	"MultiRouteNetwork/internal/nics"
	"syscall"
	"testing"
	"time"
)

func TestTransferTraceTCPCountsProgressEOFAndCloseOnce(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted := make(chan *net.TCPConn, 1)
	go func() {
		c, err := ln.Accept()
		if err == nil {
			accepted <- c.(*net.TCPConn)
		}
	}()
	raw, err := net.Dial("tcp4", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	peer := <-accepted
	defer peer.Close()
	s := NewScheduler([]nics.Nic{nic("test", "127.0.0.1", 1)})
	sl := s.Acquire(nil)
	events := make(chan TransferTrace, 10)
	c := &slotConn{TCPConn: raw.(*net.TCPConn), s: s, sl: sl, fam: Fam4, activity: s.track(sl, Fam4), transfer: newTransferMeter("test", func(v TransferTrace) { events <- v })}
	defer c.Close()
	peer.Write([]byte("download"))
	if _, err := io.ReadFull(c, make([]byte, 8)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write([]byte("upload")); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(peer, make([]byte, 6)); err != nil {
		t.Fatal(err)
	}
	select {
	case v := <-events:
		if v.Phase != "progress" || v.DownBytes != 8 || v.UpBytes != 6 || v.DownBps <= 0 {
			t.Fatalf("progress %+v", v)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no progress")
	}
	peer.CloseWrite()
	if _, err := c.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("EOF %v", err)
	}
	c.CloseWithReason("capture-ended")
	c.Close()
	v := <-events
	if v.Phase != "transfer-closed" || v.DownBytes != 8 || v.UpBytes != 6 || v.ReadState != "eof" || v.CloseReason != "capture-ended" || v.AverageDownBps <= 0 {
		t.Fatalf("final %+v", v)
	}
	select {
	case v := <-events:
		t.Fatalf("duplicate final %+v", v)
	default:
	}
	if sl.active.Load() != 0 {
		t.Fatal("changed scheduler release")
	}
}

func TestTransferTraceUDPUnblocksReadOnClose(t *testing.T) {
	raw, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	peer, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	events := make(chan TransferTrace, 10)
	c := &slotPacketConn{PacketConn: raw, sl: &Slot{Name: "test"}, transfer: newTransferMeter("test", func(v TransferTrace) { events <- v })}
	if _, err := c.WriteTo([]byte("up"), peer.LocalAddr()); err != nil {
		t.Fatal(err)
	}
	peer.WriteTo([]byte("down"), raw.LocalAddr())
	if _, _, err := c.ReadFrom(make([]byte, 10)); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { c.ReadFrom(make([]byte, 1)); close(done) }()
	// The socket deadline makes this safe even if a regression leaves a read pending.
	raw.SetReadDeadline(time.Now().Add(time.Second))
	c.CloseWithReason("capture-ended")
	<-done
	c.Close()
	var final TransferTrace
	for len(events) > 0 {
		final = <-events
	}
	if final.Phase != "transfer-closed" || final.DownBytes != 4 || final.UpBytes != 2 || final.CloseReason != "capture-ended" {
		t.Fatalf("UDP %+v", final)
	}
}

func TestTransferErrorClassificationDoesNotExposeAddresses(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{{io.EOF, "eof"}, {net.ErrClosed, "closed"}, {syscall.Errno(10054), "reset"}, {&net.OpError{Op: "read", Err: &net.DNSError{IsTimeout: true}}, "timeout"}} {
		if got := transferError(tc.err); got != tc.want {
			t.Fatalf("got %s want %s", got, tc.want)
		}
	}
}

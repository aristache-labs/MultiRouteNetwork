package tun

import (
	"context"
	"io"
	"net"
	"sync"
)

// A capture generation owns both ends of every relayed flow. Late dial results
// from a stopped generation are closed rather than attached to the next Start.
type flowSession struct {
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	closed bool
	flows  map[io.Closer]struct{}
}

type sessionConn struct {
	net.Conn
	release func()
}

func (c *sessionConn) Close() error {
	if c.release != nil {
		c.release()
	}
	return c.Conn.Close()
}
func (c *sessionConn) CloseRead() error {
	if x, ok := c.Conn.(interface{ CloseRead() error }); ok {
		return x.CloseRead()
	}
	return nil
}
func (c *sessionConn) CloseWrite() error {
	if x, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return x.CloseWrite()
	}
	return nil
}

type sessionPacketConn struct {
	net.PacketConn
	release func()
}

func (c *sessionPacketConn) Close() error {
	if c.release != nil {
		c.release()
	}
	return c.PacketConn.Close()
}
func newFlowSession() *flowSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &flowSession{ctx: ctx, cancel: cancel, flows: map[io.Closer]struct{}{}}
}
func (s *flowSession) track(c io.Closer) (func(), bool) {
	if s == nil {
		c.Close()
		return nil, false
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		c.Close()
		return nil, false
	}
	s.flows[c] = struct{}{}
	s.mu.Unlock()
	return func() { s.mu.Lock(); delete(s.flows, c); s.mu.Unlock() }, true
}
func (s *flowSession) stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	flows := s.flows
	s.flows = nil
	s.cancel()
	s.mu.Unlock()
	for c := range flows {
		c.Close()
	}
}

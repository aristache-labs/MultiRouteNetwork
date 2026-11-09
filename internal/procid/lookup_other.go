//go:build !windows

package procid

import "net/netip"

// Proc is the owner of a socket. Known is false when the platform cannot say.
type Proc struct {
	PID   uint32
	Name  string
	Path  string
	Known bool
}

// Lookup resolves a local socket. Non-Windows builds report the flow as unknown.
func Lookup(string, netip.Addr, uint16) Proc { return Proc{} }

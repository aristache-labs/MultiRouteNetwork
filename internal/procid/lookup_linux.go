//go:build linux

package procid

import (
	"net/netip"
	"os"
	"sync"
)

var (
	linuxMu    sync.Mutex
	linuxCache = ownerCache{}
)

type osProc struct{}

func (osProc) Read(path string) ([]byte, error) { return os.ReadFile(path) }
func (osProc) Link(path string) (string, error) { return os.Readlink(path) }
func (osProc) List(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// Lookup maps a local TCP or UDP endpoint to a process. Unknown stays unknown.
func Lookup(network string, addr netip.Addr, port uint16) Proc {
	linuxMu.Lock()
	defer linuxMu.Unlock()
	return LookupProc(osProc{}, linuxCache, network, addr, port)
}

// Running lists processes with a readable executable path.
func Running() ([]Proc, error) {
	procs, err := ListRunning(osProc{})
	if err != nil {
		return nil, err
	}
	if procs == nil {
		return []Proc{}, nil
	}
	return procs, nil
}

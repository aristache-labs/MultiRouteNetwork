package procid

import (
	"net/netip"
	"testing"
)

func TestMatchPrefersExactAddress(t *testing.T) {
	rows := []Row{
		{Port: 443, PID: 9},
		{Local: netip.MustParseAddr("10.0.0.2"), Port: 443, PID: 4, HasAddr: true},
	}
	pid, ok := Match(rows, netip.MustParseAddr("10.0.0.2"), 443)
	if !ok || pid != 4 {
		t.Fatalf("pid=%d ok=%v", pid, ok)
	}
}

func TestMatchPortOnlyWhenAddressDiffers(t *testing.T) {
	rows := []Row{{Port: 53, PID: 7}}
	pid, ok := Match(rows, netip.MustParseAddr("127.0.0.1"), 53)
	if !ok || pid != 7 {
		t.Fatalf("pid=%d ok=%v", pid, ok)
	}
}

func TestMatchMiss(t *testing.T) {
	if _, ok := Match(nil, netip.MustParseAddr("1.1.1.1"), 1); ok {
		t.Fatal("expected miss")
	}
}

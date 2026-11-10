package main

import (
	"MultiRouteNetwork/internal/control"
	"strings"
	"testing"
)

func TestDesktopTrayStates(t *testing.T) {
	for _, tc := range []struct {
		status        *control.Status
		label, toggle string
		starting      bool
	}{
		{nil, "Offline", "Start", false},
		{&control.Status{Running: true}, "Online", "Stop", false},
		{&control.Status{Starting: true}, "Starting…", "Starting…", true},
	} {
		v := desktopTrayView(tc.status)
		if !strings.HasPrefix(v.header, tc.label) || v.toggle != tc.toggle || v.starting != tc.starting {
			t.Fatalf("wrong tray state: %+v", v)
		}
		if len([]rune(v.tooltip)) > 63 {
			t.Fatal("tooltip exceeds native budget")
		}
	}
}
func TestDesktopTrayRatesAndAdapterOrder(t *testing.T) {
	st := &control.Status{Running: true, DownBps: 12500000, UpBps: 125000, Adapters: []control.AdapterCard{
		{Name: "Wi-Fi", DownBps: 125000, UpBps: 1250},
		{Name: "Ethernet", DownBps: 12375000},
		{Name: "Tailscale", Excluded: true},
	}}
	v := desktopTrayView(st)
	if !strings.Contains(v.header, "100.0 Mbit/s") || !strings.Contains(v.header, "1.0 Mbit/s") {
		t.Fatal(v.header)
	}
	if len(v.rows) != 3 || !strings.HasPrefix(v.rows[0], "Ethernet:") || !strings.Contains(v.rows[1], "not routed") || !strings.Contains(v.rows[2], "10 kbit/s") {
		t.Fatal(v.rows)
	}
	if st.Adapters[0].Name != "Wi-Fi" {
		t.Fatal("tray mutated shared adapter order")
	}
	st.Adapters = nil
	if len(desktopTrayView(st).rows) != 0 {
		t.Fatal("removed adapter remained visible")
	}
}

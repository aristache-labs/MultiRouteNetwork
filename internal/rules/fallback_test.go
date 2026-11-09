package rules

import (
	"testing"
	"time"
)

func TestExplicitFallbacks(t *testing.T) {
	for _, tc := range []struct {
		name       string
		fallback   Fallback
		adapters   []string
		lastResort bool
		wantPin    string
		wantBlock  bool
	}{
		{"default", FallbackDefault, nil, false, "Wi-Fi", false},
		{"ordered", FallbackChoose, []string{"Ethernet", "Wi-Fi"}, false, "Ethernet", false},
		{"ordered reversed", FallbackChoose, []string{"Wi-Fi", "Ethernet"}, false, "Wi-Fi", false},
		{"missing blocks", FallbackChoose, []string{"missing"}, false, "", true},
		{"missing last resort", FallbackChoose, []string{"missing"}, true, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := ads()
			pool[0].Default = true
			r := Rule{Exe: "a.exe", Mode: ModeOne, Adapters: []string{"Tailscale"}, Fallback: tc.fallback, FallbackAdapters: tc.adapters, LastResort: tc.lastResort}
			got := Choose([]Rule{r}, Proc{Name: "a.exe", Known: true}, pool, nil, time.Now())
			if got.Pin != tc.wantPin || got.Block != tc.wantBlock {
				t.Fatalf("%+v", got)
			}
			if tc.lastResort && len(got.Only) != 2 {
				t.Fatalf("last resort pool: %+v", got)
			}
			// A working primary always takes priority over fallback.
			r.Adapters = []string{"Ethernet"}
			got = Choose([]Rule{r}, Proc{Name: "a.exe", Known: true}, pool, nil, time.Now())
			if got.Pin != "Ethernet" {
				t.Fatalf("primary lost priority: %+v", got)
			}
		})
	}
}

func TestUnavailableDefaultFallbackBlocks(t *testing.T) {
	r := Rule{Exe: "a.exe", Mode: ModeOne, Adapters: []string{"Tailscale"}, Fallback: FallbackDefault}
	got := Choose([]Rule{r}, Proc{Name: "a.exe", Known: true}, ads(), nil, time.Now())
	if !got.Block {
		t.Fatalf("missing default must not bypass the rule: %+v", got)
	}
}

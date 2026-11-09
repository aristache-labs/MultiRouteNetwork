package rules

import (
	"testing"
	"time"
)

func ads() []Adapter {
	return []Adapter{
		{Name: "Wi-Fi", Up: true, Latency: 40 * time.Millisecond},
		{Name: "Ethernet", Up: true, Latency: 10 * time.Millisecond},
		{Name: "Tailscale", Up: false, Latency: 5 * time.Millisecond},
	}
}

func TestExactPathBeatsName(t *testing.T) {
	list := []Rule{
		{Exe: "chrome.exe", Mode: ModeOne, Adapters: []string{"Wi-Fi"}, Fallback: FallbackWait},
		{Exe: "chrome.exe", Path: `C:\chrome.exe`, ExactPath: true, Mode: ModeOne, Adapters: []string{"Ethernet"}, Fallback: FallbackWait},
	}
	got := Choose(list, Proc{Name: "chrome.exe", Path: `C:\chrome.exe`, Known: true}, ads(), nil, time.Now())
	if got.Pin != "Ethernet" {
		t.Fatalf("pin=%q", got.Pin)
	}
}

func TestWaitBlocksWhenSelectedIsDown(t *testing.T) {
	list := []Rule{{Exe: "a.exe", Mode: ModeOne, Adapters: []string{"Tailscale"}, Fallback: FallbackWait}}
	got := Choose(list, Proc{Name: "a.exe", Known: true}, ads(), nil, time.Now())
	if !got.Block {
		t.Fatal("expected block")
	}
}

func TestOthersUsesLiveAdapters(t *testing.T) {
	list := []Rule{{Exe: "a.exe", Mode: ModeOne, Adapters: []string{"Tailscale"}, Fallback: FallbackOthers}}
	got := Choose(list, Proc{Name: "a.exe", Known: true}, ads(), nil, time.Now())
	if got.Balance || got.Block || len(got.Only) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestLowestLatencyHolds(t *testing.T) {
	list := []Rule{{Exe: "a.exe", Mode: ModeAll, LowestLatency: true, Fallback: FallbackOthers}}
	var hold Hold
	now := time.Now()
	first := Choose(list, Proc{Name: "a.exe", Known: true}, ads(), &hold, now)
	if first.Pin != "Ethernet" {
		t.Fatalf("pin=%q", first.Pin)
	}
	slow := []Adapter{{Name: "Ethernet", Up: true, Latency: 80 * time.Millisecond}, {Name: "Wi-Fi", Up: true, Latency: 1 * time.Millisecond}}
	second := Choose(list, Proc{Name: "a.exe", Known: true}, slow, &hold, now.Add(time.Second))
	if second.Pin != "Ethernet" {
		t.Fatalf("flapped to %q", second.Pin)
	}
}

func TestUnknownBalances(t *testing.T) {
	got := Choose(nil, Proc{}, ads(), nil, time.Now())
	if !got.Balance {
		t.Fatal(got)
	}
}

func TestDNSPresets(t *testing.T) {
	if dnsOf(Rule{DNS: "quad9"}) != "9.9.9.9" || dnsOf(Rule{DNS: "windows"}) != "" {
		t.Fatal(dnsOf(Rule{DNS: "quad9"}))
	}
}

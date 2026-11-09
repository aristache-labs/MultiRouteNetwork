package control

import (
	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/nics"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

func TestAdapterCacheTTLHotplugAndFailure(t *testing.T) {
	now := time.Unix(100, 0)
	calls := 0
	interfaces := []net.Interface{{Name: "Ethernet", Index: 10}}
	var failure error
	cache := newAdapterIdentityCache(func() ([]net.Interface, error) { calls++; return interfaces, failure })
	cache.now = func() time.Time { return now }
	original := cache.get()
	for i := 0; i < 12; i++ {
		cache.get()
	}
	if calls != 1 {
		t.Fatalf("enumerated each read: %d", calls)
	}
	interfaces = []net.Interface{{Name: "Ethernet", Index: 20}, {Name: "Wi-Fi", Index: 30}}
	now = now.Add(adapterIdentityTTL)
	updated := cache.get()
	if updated["Ethernet"].Index != 20 || updated["Wi-Fi"].Index != 30 {
		t.Fatal("missed reindex/hotplug")
	}
	if original["Ethernet"].Index != 10 {
		t.Fatal("mutated published cache map")
	}
	interfaces = []net.Interface{{Name: "Wi-Fi", Index: 30}}
	now = now.Add(adapterIdentityTTL)
	if _, found := cache.get()["Ethernet"]; found {
		t.Fatal("retained removed adapter")
	}
	failure = errors.New("enumeration unavailable")
	now = now.Add(adapterIdentityTTL)
	if len(cache.get()) != 0 {
		t.Fatal("used stale indices after failure")
	}
	before := calls
	cache.get()
	cache.get()
	if calls != before {
		t.Fatal("retried failed enumeration at telemetry cadence")
	}
	failure = nil
	now = now.Add(time.Second)
	if cache.get()["Wi-Fi"].Index != 30 {
		t.Fatal("did not recover after retry interval")
	}
}

func TestAdapterCacheConcurrentReadersEnumerateOnce(t *testing.T) {
	calls := 0
	cache := newAdapterIdentityCache(func() ([]net.Interface, error) { calls++; return []net.Interface{{Name: "Ethernet", Index: 10}}, nil })
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if cache.get()["Ethernet"].Index != 10 {
				t.Error("missing adapter")
			}
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("concurrent refresh stampede: %d", calls)
	}
}

func TestCachedIdentityKeepsQuarterSecondCountersAndGraphs(t *testing.T) {
	s := New("", nil)
	sw := &testSwitch{}
	sw.on.Store(true)
	s.sw = sw
	now := time.Unix(100, 0)
	s.adapterCache.now = func() time.Time { return now }
	enumerations, counterReads := 0, 0
	s.interfaces = func() ([]net.Interface, error) {
		enumerations++
		return []net.Interface{{Name: "Ethernet", Index: 10}, {Name: "MultiRouteNetwork", Index: 11}}, nil
	}
	s.octets = func(index int) (uint64, uint64, bool) {
		if index != 10 {
			t.Fatalf("read hidden/unused counter %d", index)
		}
		counterReads++
		return uint64(counterReads * 1000), uint64(counterReads * 500), true
	}
	snap := []engine.SlotStats{{Name: "Ethernet", Present: true}}
	skipped := []nics.Skip{{Name: "MultiRouteNetwork", Reason: "virtual adapter"}}
	s.Sample(snap, skipped, 250*time.Millisecond) // Baseline, not a full interval.
	for i := 0; i < 12; i++ {
		now = now.Add(250 * time.Millisecond)
		s.Sample(snap, skipped, 250*time.Millisecond)
		// Status and sampling share the same interface cache.
		status := s.Snapshot(nil, nil, true)
		if status.DownBps != 4000 || status.UpBps != 2000 {
			t.Fatalf("wrong quarter-second rates: %v/%v", status.DownBps, status.UpBps)
		}
	}
	if counterReads != 13 || enumerations != 2 {
		t.Fatalf("counter reads=%d enumerations=%d", counterReads, enumerations)
	}
	if s.uses["Ethernet"].todayDown != 12000 || s.uses["Ethernet"].todayUp != 6000 {
		t.Fatal("changed accumulated usage")
	}
	g := s.graphs["Ethernet"]
	if len(g.down) != 240 {
		t.Fatal("changed 60-second history length")
	}
	for _, v := range g.down[len(g.down)-12:] {
		if v != 4000 {
			t.Fatal("skipped quarter-second graph sample")
		}
	}
}

func TestCounterBaselineResetsOnReindexOrFailure(t *testing.T) {
	s := New("", nil)
	s.sampleSeconds = 0.25
	read := func(index int, in uint64, ok bool) float64 {
		down, _ := s.sampleLink("Ethernet", linkReading{index: index, in: in, ok: ok})
		return down
	}
	if read(10, 1000, true) != 0 || read(10, 2000, true) != 4000 {
		t.Fatal("initial baseline/rate")
	}
	if read(20, 1000000, true) != 0 {
		t.Fatal("reindex generated false usage spike")
	}
	if read(20, 1001000, true) != 4000 {
		t.Fatal("did not resume quarter-second rates")
	}
	if read(20, 0, false) != 0 {
		t.Fatal("failed read retained stale rate")
	}
	if read(20, 2000000, true) != 0 || read(20, 2001000, true) != 4000 {
		t.Fatal("failure recovery generated false usage")
	}
	if down, _ := s.sampleLink("Ethernet", linkReading{index: 20, in: 3000000, ok: true, hardware: "replacement-device"}); down != 0 {
		t.Fatal("reused index on a replacement device generated false usage")
	}
}

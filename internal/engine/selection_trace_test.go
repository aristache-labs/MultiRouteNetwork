package engine

import (
	"MultiRouteNetwork/internal/nics"
	"testing"
	"time"
)

func TestSelectionTraceExplainsScoresAndExclusions(t *testing.T) {
	s := NewScheduler([]nics.Nic{nic("Ethernet", "127.0.0.1", 1), nic("Wi-Fi", "127.0.0.2", 0.1), nic("Phone", "127.0.0.3", 1)})
	for _, sl := range s.slots {
		sl.state[Fam4] = stateUp
	}
	s.slots[2].downUntil[Fam4] = time.Now().Add(time.Minute)
	var got SelectionTrace
	chosen := s.AcquireForHint(Fam4, nil, FlowHint{Trace: func(v SelectionTrace) { s.Snapshot(); got = v }})
	if chosen.Name != "Ethernet" || got.Selected != "Ethernet" {
		t.Fatalf("selection %+v", got)
	}
	wifi := got.Candidates[1]
	if wifi.Score == nil || *wifi.Score != 10 || wifi.Weight != 0.1 || wifi.Reason != "score-or-tie" {
		t.Fatalf("Wi-Fi score %+v", wifi)
	}
	if got.Candidates[2].Reason != "health-tier" || got.Candidates[2].CooldownMs <= 0 {
		t.Fatalf("cooldown %+v", got)
	}
	s.AcquireForHint(Fam4, map[string]struct{}{"Ethernet": {}}, FlowHint{Trace: func(v SelectionTrace) { got = v }})
	if got.Candidates[0].Reason != "retry-excluded" || got.Selected != "Wi-Fi" {
		t.Fatalf("retry %+v", got)
	}
	s.SetPolicy(func(FlowHint, []AdapterView) Decision { return Decision{Only: []string{"Wi-Fi"}} })
	s.AcquireForHint(Fam4, nil, FlowHint{Trace: func(v SelectionTrace) { got = v }})
	if got.Policy != "subset" || got.Selected != "Wi-Fi" || got.Candidates[0].Reason != "policy-excluded" {
		t.Fatalf("policy %+v", got)
	}
	s.AcquireForHint(Fam6, nil, FlowHint{Trace: func(v SelectionTrace) { got = v }})
	if got.Outcome != "no-candidate" || got.Candidates[1].Reason != "no-family-address" {
		t.Fatalf("IPv6 %+v", got)
	}
}

func TestSelectionTracingPreservesWeightedDistribution(t *testing.T) {
	for _, tracing := range []bool{false, true} {
		s := NewScheduler([]nics.Nic{nic("A", "127.0.0.1", 2), nic("B", "127.0.0.2", 1)})
		hint := FlowHint{}
		events := 0
		if tracing {
			hint.Trace = func(SelectionTrace) { events++ }
		}
		counts := map[string]int{}
		for i := 0; i < 60; i++ {
			counts[s.AcquireForHint(Fam4, nil, hint).Name]++
		}
		if counts["A"] != 40 || counts["B"] != 20 {
			t.Fatalf("tracing=%v: %v", tracing, counts)
		}
		if tracing && events != 60 {
			t.Fatal("missing diagnostics")
		}
	}
}

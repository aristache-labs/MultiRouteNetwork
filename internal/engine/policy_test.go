package engine

import (
	"net"
	"testing"

	"MultiRouteNetwork/internal/nics"
)

func TestPolicyPin(t *testing.T) {
	s := NewScheduler([]nics.Nic{
		{Name: "a", IP: net.ParseIP("10.0.0.1"), Weight: 1},
		{Name: "b", IP: net.ParseIP("10.0.0.2"), Weight: 1},
	})
	s.SetPolicy(func(FlowHint, []AdapterView) Decision { return Decision{Pin: "b"} })
	sl := s.Acquire(nil)
	if sl == nil || sl.Name != "b" {
		t.Fatalf("got %#v", sl)
	}
	s.Release(sl)
}

func TestPolicyBlock(t *testing.T) {
	s := NewScheduler([]nics.Nic{{Name: "a", IP: net.ParseIP("10.0.0.1"), Weight: 1}})
	s.SetPolicy(func(FlowHint, []AdapterView) Decision { return Decision{Block: true} })
	if sl := s.Acquire(nil); sl != nil {
		t.Fatal("expected no adapter")
	}
}

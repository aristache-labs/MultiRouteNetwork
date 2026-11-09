package control

import (
	"net"
	"net-merge/internal/engine"
	"net-merge/internal/nics"
	"net-merge/internal/rules"
	"net-merge/internal/store"
	"testing"
)

func TestRoutingModesKeepWeightedBalancing(t *testing.T) {
	for _, mode := range []string{"all", "several", "unknown", "unmatched"} {
		t.Run(mode, func(t *testing.T) {
			s := New("", nil)
			s.PutSettings(store.Settings{DefaultAdapter: "A"})
			r := rules.Rule{Exe: "app.exe", Mode: rules.ModeAll, Preferred: "A", Fallback: rules.FallbackWait}
			if mode == "several" {
				r.Mode = rules.ModeSeveral
				r.Adapters = []string{"A", "B"}
			}
			s.PutRules([]rules.Rule{r})
			sched := engine.NewScheduler([]nics.Nic{{Name: "A", IP: net.ParseIP("127.0.0.1"), Weight: 2}, {Name: "B", IP: net.ParseIP("127.0.0.2"), Weight: 1}})
			sched.SetAuto(true)
			s.Attach(sched, &testSwitch{})
			hint := engine.FlowHint{Name: "app.exe", Known: true}
			if mode == "unknown" {
				hint.Known = false
			}
			if mode == "unmatched" {
				hint.Name = "other.exe"
			}
			counts := map[string]int{}
			for i := 0; i < 12; i++ {
				slot := sched.AcquireForHint(engine.Fam4, nil, hint)
				if slot == nil {
					t.Fatal("blocked")
				}
				counts[slot.Name]++
			}
			if counts["A"] != 8 || counts["B"] != 4 {
				t.Fatalf("weighted balance lost: %v", counts)
			}
		})
	}
}

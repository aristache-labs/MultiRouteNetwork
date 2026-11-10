package control

import (
	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/rules"
	"math"
	"testing"
	"time"
)

func TestSeventyMillisecondTelemetryRatesAndTotals(t *testing.T) {
	s := New("", nil)
	sw := &testSwitch{}
	sw.On()
	s.Attach(engine.NewScheduler(nil), sw)
	s.PutRules([]rules.Rule{{Exe: "app.exe"}})
	snap := []engine.SlotStats{{Name: "test", Present: true, LinkIn: 1000000}}
	for i := 0; i < 10; i++ {
		s.account("app.exe", "", "test", rules.Rule{}, true, 70000, false)
		s.tick(snap, nil, 70*time.Millisecond, false)
	}
	st := s.Snapshot(nil, nil, true)
	if math.Abs(st.Apps[0].DownBps-1000000) > 1e-8 || st.Apps[0].DownBytes != 700000 || st.Adapters[0].TodayDown != 700000 {
		t.Fatalf("rates/totals: rate=%v app=%v adapter=%v", st.Apps[0].DownBps, st.Apps[0].DownBytes, st.Adapters[0].TodayDown)
	}
	if len(st.Down) != 858 || len(st.Apps[0].Down) != 858 {
		t.Fatal("graph must retain 858 seventy-millisecond samples")
	}
	s.account("app.exe", "", "test", rules.Rule{}, true, 1000, false)
	s.tick(snap, nil, 400*time.Millisecond, false)
	if got := s.Snapshot(nil, nil, true).Apps[0].DownBps; got != 2500 {
		t.Fatalf("elapsed time ignored: %v", got)
	}
}

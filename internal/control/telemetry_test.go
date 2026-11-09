package control

import (
	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/rules"
	"testing"
	"time"
)

func TestQuarterSecondTelemetryRatesAndTotals(t *testing.T) {
	s := New("", nil)
	sw := &testSwitch{}
	sw.On()
	s.Attach(engine.NewScheduler(nil), sw)
	s.PutRules([]rules.Rule{{Exe: "app.exe"}})
	snap := []engine.SlotStats{{Name: "test", Present: true, LinkIn: 1000000}}
	for i := 0; i < 4; i++ {
		s.account("app.exe", "", "test", rules.Rule{}, true, 250000, false)
		s.tick(snap, nil, 250*time.Millisecond, false)
	}
	st := s.Snapshot(nil, nil, true)
	if st.Apps[0].DownBps != 1000000 || st.Apps[0].DownBytes != 1000000 || st.Adapters[0].TodayDown != 1000000 {
		t.Fatalf("rates/totals: %+v", st)
	}
	if len(st.Down) != 240 || len(st.Apps[0].Down) != 240 {
		t.Fatal("graph must retain 240 quarter-second samples")
	}
	s.account("app.exe", "", "test", rules.Rule{}, true, 1000, false)
	s.tick(snap, nil, 400*time.Millisecond, false)
	if got := s.Snapshot(nil, nil, true).Apps[0].DownBps; got != 2500 {
		t.Fatalf("elapsed time ignored: %v", got)
	}
}

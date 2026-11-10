package control

import (
	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/rules"
	"MultiRouteNetwork/internal/store"
	"testing"
	"time"
)

func TestRemovingDataCapUnblocksWithoutRestart(t *testing.T) {
	for _, removeRule := range []bool{false, true} {
		t.Run(map[bool]string{false: "clear cap", true: "delete rule"}[removeRule], func(t *testing.T) {
			s := New("", nil)
			sw := &testSwitch{}
			sw.On()
			s.Attach(engine.NewScheduler(nil), sw)
			// Unknown allows exercising the real flow Stop callback without OS lookups.
			old := rules.Rule{ID: "cap", Exe: "Unknown", CapGB: 0.000001, CapPeriod: "month"}
			if err := s.PutRules([]rules.Rule{old}); err != nil {
				t.Fatal(err)
			}
			h := s.Prepare("tcp", "invalid", "", 0, 0)
			h.OnBytes("test", true, 1000, false)
			if !h.Stop() {
				t.Fatal("cap not reached")
			}
			updated := old
			updated.CapGB = 0
			list := []rules.Rule{updated}
			if removeRule {
				list = nil
			}
			if err := s.PutRules(list); err != nil {
				t.Fatal(err)
			}
			if h.Stop() {
				t.Fatal("flow remains capped after rule update")
			}
			if s.apps["Unknown"].warn {
				t.Fatal("stale warning")
			}
			// Existing callbacks captured old, capped rules before the edit.
			h.OnBytes("test", true, 100, false)
			if h.Stop() {
				t.Fatal("old flow callback re-enabled removed cap")
			}
			if s.apps["Unknown"].down != 1100 {
				t.Fatal("usage counters changed")
			}
		})
	}
}

func TestEditingCapReevaluatesPersistedAndPendingUsage(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.AddSample(time.Now(), "chrome.exe", "", "test", 900, 0); err != nil {
		t.Fatal(err)
	}
	s := New("", db)
	sw := &testSwitch{}
	sw.On()
	s.Attach(engine.NewScheduler(nil), sw)
	old := rules.Rule{ID: "chrome", Exe: "chrome.exe", CapGB: 0.000001, CapPeriod: "month"}
	if err := s.PutRules([]rules.Rule{old}); err != nil {
		t.Fatal(err)
	}
	s.account("chrome.exe", "", "test", old, true, 100, false)
	if !s.apps["chrome.exe"].capped {
		t.Fatal("cap not reached")
	}
	raised := old
	raised.CapGB = 0.000002
	if err := s.PutRules([]rules.Rule{raised}); err != nil {
		t.Fatal(err)
	}
	if s.apps["chrome.exe"].capped || s.apps["chrome.exe"].warn {
		t.Fatal("raised cap did not clear flags")
	}
	s.account("chrome.exe", "", "test", old, true, 100, false)
	if s.apps["chrome.exe"].capped {
		t.Fatal("captured cap re-blocked application")
	}
	lowered := old
	lowered.CapGB = 0.0000005
	if err := s.PutRules([]rules.Rule{lowered}); err != nil {
		t.Fatal(err)
	}
	if !s.apps["chrome.exe"].capped {
		t.Fatal("lowered cap ignored usage")
	}
	if s.apps["chrome.exe"].period != 1100 {
		t.Fatalf("lost pending usage: %d", s.apps["chrome.exe"].period)
	}
	uncapped := old
	uncapped.CapGB = 0
	if err := s.PutRules([]rules.Rule{uncapped}); err != nil {
		t.Fatal(err)
	}
	s.account("chrome.exe", "", "test", old, true, 100, false)
	if s.apps["chrome.exe"].capped {
		t.Fatal("removed cap reactivated")
	}
	if err := s.PutRules([]rules.Rule{raised}); err != nil {
		t.Fatal(err)
	}
	if s.apps["chrome.exe"].period != 1200 {
		t.Fatal("re-enabling cap lost or duplicated usage")
	}
	s.Flush()
	if err := s.PutRules([]rules.Rule{raised}); err != nil {
		t.Fatal(err)
	}
	if s.apps["chrome.exe"].period != 1200 {
		t.Fatal("flush duplicated cap usage")
	}
}

func TestCapEditsPreserveExactPathPriority(t *testing.T) {
	s := New("", nil)
	sw := &testSwitch{}
	sw.On()
	s.Attach(engine.NewScheduler(nil), sw)
	base := rules.Rule{ID: "base", Exe: "chrome.exe", CapGB: 0.0000005, CapPeriod: "month"}
	exact := rules.Rule{ID: "path", Exe: "chrome.exe", Path: `C:\Chrome\chrome.exe`, ExactPath: true, CapGB: 0.000001, CapPeriod: "month"}
	if err := s.PutRules([]rules.Rule{base, exact}); err != nil {
		t.Fatal(err)
	}
	s.account("chrome.exe", exact.Path, "test", exact, true, 1000, false)
	if !s.apps["chrome.exe"].capped {
		t.Fatal("path cap not reached")
	}
	exact.CapGB = 0
	if err := s.PutRules([]rules.Rule{base, exact}); err != nil {
		t.Fatal(err)
	}
	if s.apps["chrome.exe"].capped {
		t.Fatal("exe cap overrode uncapped exact path")
	}
	if err := s.PutRules([]rules.Rule{base}); err != nil {
		t.Fatal(err)
	}
	if !s.apps["chrome.exe"].capped {
		t.Fatal("deleting path rule bypassed capped exe fallback")
	}
}

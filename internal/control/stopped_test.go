package control

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"net-merge/internal/engine"
	"net-merge/internal/rules"
	"net-merge/internal/store"
)

type testSwitch struct{ on atomic.Bool }

func (s *testSwitch) On() error  { s.on.Store(true); return nil }
func (s *testSwitch) Off()       { s.on.Store(false) }
func (s *testSwitch) IsOn() bool { return s.on.Load() }

func TestStopFreezesStatisticsAndResumeSkipsPausedTraffic(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s := New(t.TempDir(), db)
	sw := &testSwitch{}
	sw.On()
	s.Attach(engine.NewScheduler(nil), sw)
	s.PutRules([]rules.Rule{{ID: "app", Exe: "chrome.exe"}})
	snap := []engine.SlotStats{{Name: "Wi-Fi", Present: true, LinkIn: 1000}}
	s.account("chrome.exe", "", "Wi-Fi", rules.Rule{}, true, 100, false)
	s.Tick(snap, nil)
	s.Stop()
	s.account("chrome.exe", "", "Wi-Fi", rules.Rule{}, true, 500, false)
	s.Tick(snap, nil)
	s.Flush()
	st := s.Snapshot(nil, nil, false)
	if st.DownBps != 0 || st.Adapters[0].DownBps != 0 || st.Apps[0].DownBps != 0 {
		t.Fatalf("stopped rates: %+v", st)
	}
	if st.Adapters[0].TodayDown != 1000 || st.Apps[0].DownBytes != 100 {
		t.Fatalf("stopped counters grew: %+v", st)
	}
	rows, err := s.Usage(time.Unix(0, 0))
	if err != nil || len(rows) != 1 || rows[0].Down != 100 {
		t.Fatalf("usage: %+v, %v", rows, err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	s.Tick(snap, nil) // The first sample may overlap the paused interval.
	if got := s.Snapshot(nil, nil, true).Adapters[0].TodayDown; got != 1000 {
		t.Fatalf("resume counted paused traffic: %d", got)
	}
	s.Tick(snap, nil)
	if got := s.Snapshot(nil, nil, true).Adapters[0].TodayDown; got != 2000 {
		t.Fatalf("resume did not record: %d", got)
	}
}

func TestStoppedSnapshotLoadsSavedAppsWithoutEngine(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	db.SaveRules([]rules.Rule{{ID: "saved", Exe: "chrome.exe"}})
	db.SaveSettings(store.Settings{Theme: "light"})
	db.Close()
	st := LocalStatus(dir)
	if st.Running || st.Theme != "light" || len(st.Apps) != 1 || st.Apps[0].Exe != "chrome.exe" {
		t.Fatalf("offline status: %+v", st)
	}
	if st.DownBps != 0 || st.Apps[0].DownBytes != 0 {
		t.Fatal("offline discovery recorded traffic")
	}
	// Reading a missing store must not create a database.
	LocalStatus(filepath.Join(dir, "missing"))
	if _, err := os.Stat(filepath.Join(dir, "missing")); !os.IsNotExist(err) {
		t.Fatalf("offline discovery created state: %v", err)
	}
}

func TestLocalHistoryWithoutEngine(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddSample(time.Now(), "chrome.exe", "", "Wi-Fi", 100, 40); err != nil {
		t.Fatal(err)
	}
	db.Close()
	reply, err := LocalHistory(dir, Request{Op: "usage", Range: "today"})
	if err != nil || !reply.OK || len(reply.Usage) != 1 || reply.Usage[0].Down != 100 {
		t.Fatalf("offline usage: %+v, %v", reply, err)
	}
	if reply.Usage[0].ByNICDown["Wi-Fi"] != 100 || reply.Usage[0].ByNICUp["Wi-Fi"] != 40 {
		t.Fatalf("offline directional totals: %+v", reply.Usage[0])
	}
	reply, err = LocalHistory(dir, Request{Op: "csv", Range: "today"})
	if err != nil || !reply.OK || reply.CSV == "" {
		t.Fatalf("offline CSV: %+v, %v", reply, err)
	}
}

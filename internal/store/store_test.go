package store

import (
	"path/filepath"
	"testing"
	"time"

	"net-merge/internal/rules"
)

func TestRulesAndUsageRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	in := []rules.Rule{{ID: "1", Exe: "chrome.exe", Mode: rules.ModeOne, Adapters: []string{"Wi-Fi"}, Fallback: rules.FallbackWait}}
	if err := s.SaveRules(in); err != nil {
		t.Fatal(err)
	}
	out, err := s.Rules()
	if err != nil || len(out) != 1 || out[0].Exe != "chrome.exe" {
		t.Fatalf("%v %+v", err, out)
	}
	now := time.Now()
	if err := s.AddSample(now, "chrome.exe", `C:\chrome.exe`, "Wi-Fi", 100, 40); err != nil {
		t.Fatal(err)
	}
	rows, err := s.Usage(now.Add(-time.Minute))
	if err != nil || len(rows) != 1 || rows[0].Down != 100 || rows[0].ByNIC["Wi-Fi"] != 140 {
		t.Fatalf("%v %+v", err, rows)
	}
	if rows[0].ByNICDown["Wi-Fi"] != 100 || rows[0].ByNICUp["Wi-Fi"] != 40 {
		t.Fatalf("directional adapter totals: %+v", rows[0])
	}
	st := DefaultSettings()
	st.Theme = "light"
	if err := s.SaveSettings(st); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadSettings()
	if err != nil || got.Theme != "light" || !got.StartRouting {
		t.Fatalf("%v %+v", err, got)
	}
}

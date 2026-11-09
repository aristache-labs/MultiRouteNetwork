package control

import (
	"math"
	"MultiRouteNetwork/internal/rules"
	"MultiRouteNetwork/internal/store"
	"testing"
	"time"
)

func TestAdapterLimitsAreSharedAndDirectional(t *testing.T) {
	s := New("", nil)
	s.PutSettings(store.Settings{AdapterDown: map[string]float64{"Wi-Fi": 1}, AdapterUp: map[string]float64{"Wi-Fi": 10}})
	app := func() *appMeter { return &appMeter{buckets: map[string]*bucket{}} }
	s.paceLocked(app(), rules.Rule{}, "Wi-Fi", true, 16384)
	wait := s.paceLocked(app(), rules.Rule{}, "Wi-Fi", true, 16384)
	if wait < 50*time.Millisecond {
		t.Fatalf("second app got a separate allowance: %v", wait)
	}
	if got := s.paceLocked(app(), rules.Rule{}, "Wi-Fi", false, 1000); got != 0 {
		t.Fatal("download consumed upload allowance")
	}
	if got := s.paceLocked(app(), rules.Rule{}, "Ethernet", true, 16384); got != 0 {
		t.Fatal("limited an unrelated adapter")
	}
	if err := s.PutSettings(store.Settings{AdapterDown: map[string]float64{"Wi-Fi": math.NaN()}}); err == nil {
		t.Fatal("accepted invalid limit")
	}
	if err := s.PutSettings(store.Settings{}); err != nil {
		t.Fatal(err)
	}
	if got := s.paceLocked(app(), rules.Rule{}, "Wi-Fi", true, 16384); got != 0 {
		t.Fatal("clearing limit did not take effect")
	}
}
func TestSettingsSaveWithoutEngine(t *testing.T) {
	dir := t.TempDir()
	rep, err := LocalRules(dir, Request{Op: "putSettings", Settings: store.Settings{StartRouting: false, AdapterDown: map[string]float64{"Wi-Fi": 5}, AdapterUp: map[string]float64{"Wi-Fi": 2}, AutoUpdate: true}})
	if err != nil || !rep.OK {
		t.Fatal(err)
	}
	got := LocalStatus(dir).Settings
	if got.StartRouting || got.AdapterDown["Wi-Fi"] != 5 || got.AdapterUp["Wi-Fi"] != 2 || got.AutoUpdate {
		t.Fatalf("bad persisted settings: %+v", got)
	}
}

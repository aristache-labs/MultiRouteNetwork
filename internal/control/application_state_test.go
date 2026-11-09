package control

import (
	"net-merge/internal/procid"
	"net-merge/internal/rules"
	"testing"
)

func TestAppRunningMatchesProcessRatherThanTraffic(t *testing.T) {
	ps := []procid.Proc{{Name: "chrome.exe", Path: `C:\Apps\chrome.exe`}}
	if !appRunning(rules.Rule{Exe: "Chrome.exe"}, ps) {
		t.Fatal("case insensitive process not found")
	}
	if appRunning(rules.Rule{Exe: "missing.exe"}, ps) {
		t.Fatal("missing process running")
	}
	if !appRunning(rules.Rule{ExactPath: true, Path: `c:\apps\chrome.exe`}, ps) {
		t.Fatal("exact path not found")
	}
	if appRunning(rules.Rule{ExactPath: true, Path: `C:\Other\chrome.exe`}, ps) {
		t.Fatal("wrong exact path matched")
	}
}
func TestUsingOrderFollowsTrafficOnlyOnOpenAdapters(t *testing.T) {
	s := New("", nil)
	w := s.trackAdapter("app.exe", "Wi-Fi")
	defer w()
	e := s.trackAdapter("app.exe", "Ethernet")
	defer e()
	s.trafficRates = map[string]map[string]int64{"app.exe": {"Wi-Fi": 100, "Ethernet": 10, "closed": 999}}
	if got := s.openAdapterNames("app.exe"); got != "Wi-Fi, Ethernet" {
		t.Fatal(got)
	}
	s.trafficRates["app.exe"]["Ethernet"] = 200
	if got := s.openAdapterNames("app.exe"); got != "Ethernet, Wi-Fi" {
		t.Fatal(got)
	}
	e()
	if got := s.openAdapterNames("app.exe"); got != "Wi-Fi" {
		t.Fatal(got)
	}
}

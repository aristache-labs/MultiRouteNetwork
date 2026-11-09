package control

import (
	"MultiRouteNetwork/internal/rules"
	"testing"
)

func TestUsingTracksOpenAdaptersInStableOrder(t *testing.T) {
	s := New("", nil)
	s.PutRules([]rules.Rule{{Exe: "chrome.exe"}})
	wifi := s.trackAdapter("chrome.exe", "Wi-Fi")
	eth := s.trackAdapter("chrome.exe", "Ethernet 4")
	wifi2 := s.trackAdapter("chrome.exe", "Wi-Fi")
	check := func(want string) {
		t.Helper()
		for i := 0; i < 10; i++ {
			if got := s.Snapshot(nil, nil, true).Apps[0].Via; got != want {
				t.Fatalf("got %q want %q", got, want)
			}
		}
	}
	check("Ethernet 4, Wi-Fi")
	wifi()
	wifi()
	check("Ethernet 4, Wi-Fi")
	wifi2()
	check("Ethernet 4")
	eth()
	check("")
	old := s.trackAdapter("chrome.exe", "Wi-Fi")
	s.Stop()
	fresh := s.trackAdapter("chrome.exe", "Wi-Fi")
	old()
	check("Wi-Fi")
	fresh()
	check("")
}

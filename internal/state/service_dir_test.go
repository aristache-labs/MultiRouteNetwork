package state

import (
	"runtime"
	"testing"
)

func TestServiceAndPreferenceDirectories(t *testing.T) {
	switch runtime.GOOS {
	case "linux":
		if ServiceDir() != "/var/lib/multiroute" {
			t.Fatal(ServiceDir())
		}
	case "darwin":
		if ServiceDir() != "/Library/Application Support/dev.aristache.multiroute" {
			t.Fatal(ServiceDir())
		}
	default:
		if ServiceDir() != DefaultDir() {
			t.Fatal(ServiceDir())
		}
	}
	if runtime.GOOS == "windows" {
		if PreferenceDir() != DefaultDir() {
			t.Fatal(PreferenceDir())
		}
		return
	}
	if PreferenceDir() == ServiceDir() {
		t.Fatal("window preferences use the privileged directory")
	}
}

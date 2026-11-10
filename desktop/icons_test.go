package main

import "testing"

func TestExecutablePathAllowsLocalApps(t *testing.T) {
	if !executablePathAllowed("linux", "/usr/bin/curl") || !executablePathAllowed("darwin", "/Applications/Safari.app/Contents/MacOS/Safari") {
		t.Fatal("local applications were rejected")
	}
	if executablePathAllowed("windows", `C:\Windows\win.ini`) || !executablePathAllowed("windows", `C:\Windows\explorer.exe`) {
		t.Fatal("windows extension rule changed")
	}
	for _, path := range []string{"relative", `\\server\share\app.exe`, "https://example/app", ""} {
		if executablePathAllowed("linux", path) || executablePathAllowed("windows", path) {
			t.Fatal(path)
		}
	}
}

//go:build windows

package main

import (
	"bytes"
	"image/png"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestExecutableIconReadsResourceWithoutLaunching(t *testing.T) {
	path := filepath.Join(os.Getenv("SystemRoot"), "explorer.exe")
	w := httptest.NewRecorder()
	serveAppIcon(w, httptest.NewRequest("GET", "/api/icon?path="+url.QueryEscape(path), nil))
	if w.Code != 200 {
		t.Fatalf("icon response %d", w.Code)
	}
	img, err := png.Decode(bytes.NewReader(w.Body.Bytes()))
	if err != nil || img.Bounds().Dx() != 48 || img.Bounds().Dy() != 48 {
		t.Fatalf("icon decode: %v", err)
	}
}

func TestIconRejectsNetworkAndNonExecutablePaths(t *testing.T) {
	for _, path := range []string{`\\server\share\app.exe`, `\\?\C:\app.exe`, "relative.exe", `C:\Windows\win.ini`} {
		w := httptest.NewRecorder()
		serveAppIcon(w, httptest.NewRequest("GET", "/api/icon?path="+url.QueryEscape(path), nil))
		if w.Code != 404 {
			t.Fatalf("%q: %d", path, w.Code)
		}
	}
}

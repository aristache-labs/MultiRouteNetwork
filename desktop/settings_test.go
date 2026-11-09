package main

import (
	"net"
	"MultiRouteNetwork/internal/control"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStartupSettingCanBeSavedWithoutEngine(t *testing.T) {
	calls := []bool{}
	a := &API{stateDir: t.TempDir(), connect: func() (net.Conn, error) { return nil, absentPipeError() }, engineAlive: func() bool { return false }, configureStartup: func(enabled bool) error { calls = append(calls, enabled); return nil }}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, httptest.NewRequest("POST", "/api/settings", strings.NewReader(`{"theme":"dark","startWindows":true,"startRouting":false}`)))
	if w.Code != 200 || len(calls) != 1 || !calls[0] {
		t.Fatalf("%d %s calls %v", w.Code, w.Body.String(), calls)
	}
	settings := control.LocalStatus(a.stateDir).Settings
	if !settings.StartWindows || settings.StartRouting {
		t.Fatalf("settings: %+v", settings)
	}
}

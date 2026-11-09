package main

import (
	"errors"
	"net"
	"testing"

	"net-merge/internal/control"
	"net-merge/internal/rules"
)

func TestAddEditDeleteRuleWithoutStartingEngine(t *testing.T) {
	a := &API{stateDir: t.TempDir(), connect: func() (net.Conn, error) { return nil, absentPipeError() }, engineAlive: func() bool { return false }}
	rule := rules.Rule{ID: "offline", Exe: "chrome.exe", Mode: rules.ModeAll, Fallback: rules.FallbackWait}
	if rep, err := a.call(control.Request{Op: "putRule", Rule: rule}); err != nil || !rep.OK {
		t.Fatalf("add while stopped: %+v, %v", rep, err)
	}
	st := control.LocalStatus(a.stateDir)
	if st.Running || len(st.Apps) != 1 || st.Apps[0].Exe != "chrome.exe" {
		t.Fatalf("saved app missing: %+v", st)
	}
	rule.Mode, rule.Adapters = rules.ModeOne, []string{"Wi-Fi"}
	if _, err := a.call(control.Request{Op: "putRule", Rule: rule}); err != nil {
		t.Fatal(err)
	}
	if got := control.LocalStatus(a.stateDir).Rules[0].Mode; got != rules.ModeOne {
		t.Fatal(got)
	}
	if _, err := a.call(control.Request{Op: "deleteRule", ID: rule.ID}); err != nil {
		t.Fatal(err)
	}
	if got := control.LocalStatus(a.stateDir); len(got.Apps) != 0 {
		t.Fatalf("deleted app remains: %+v", got.Apps)
	}
}

func TestBusyOrLiveEngineNeverFallsBackToLocalWrites(t *testing.T) {
	for _, live := range []bool{false, true} {
		busy := errors.New("pipe is busy")
		a := &API{stateDir: t.TempDir(), connect: func() (net.Conn, error) {
			if live {
				return nil, absentPipeError()
			}
			return nil, busy
		}, engineAlive: func() bool { return live }}
		_, err := a.call(control.Request{Op: "putRule", Rule: rules.Rule{Exe: "test.exe"}})
		if err == nil {
			t.Fatal("unavailable live engine must not silently write a local rule")
		}
		if got := control.LocalStatus(a.stateDir); len(got.Apps) != 0 {
			t.Fatal("unexpected local write")
		}
	}
}

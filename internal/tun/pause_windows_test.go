package tun

import (
	"errors"
	"golang.org/x/sys/windows"
	"testing"
)

// Exercise ordering and rollback with fake IP Helper calls; never change OS routes.
func TestPauseRouteRowsAndResumeRollback(t *testing.T) {
	rows := []windows.MibIpForwardRow2{{InterfaceIndex: 42}, {InterfaceIndex: 42}}
	rows[0].Metric = 1
	rows[1].Metric = 2
	var events []string
	fail := false
	change := func(add bool, r *windows.MibIpForwardRow2) error {
		if add {
			events = append(events, "add")
			if fail && r.Metric == 2 {
				return errors.New("create failed")
			}
		} else {
			events = append(events, "delete")
		}
		return nil
	}
	resume, err := pauseRouteRows(42, rows, change, func(int) { events = append(events, "prepare") }, func() { events = append(events, "restore") })
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[2] != "restore" {
		t.Fatalf("pause ordering %v", events)
	}
	events = nil
	if err := resume(); err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0] != "prepare" {
		t.Fatalf("resume ordering %v", events)
	}
	events = nil
	fail = true
	if err := resume(); err == nil {
		t.Fatal("ignored route failure")
	}
	want := []string{"prepare", "add", "add", "delete", "restore"}
	if len(events) != len(want) {
		t.Fatalf("rollback %v", events)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("rollback %v", events)
		}
	}
	if _, err := pauseRouteRows(42, nil, change, func(int) {}, func() {}); err == nil {
		t.Fatal("accepted missing capture routes")
	}
}

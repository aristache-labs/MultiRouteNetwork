//go:build windows

package control

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/rules"
	"MultiRouteNetwork/internal/store"
)

func testPipePath() string { return fmt.Sprintf(`\\.\pipe\MultiRoute-test-%d`, time.Now().UnixNano()) }

func TestMissingPipeIsNotReportedAsBusy(t *testing.T) {
	_, err := dialPipe(testPipePath())
	if !IsPipeAbsent(err) {
		t.Fatalf("expected missing pipe, got %v", err)
	}
}

func TestWindowsPipeServesRulesWhileStoppedAndPolled(t *testing.T) {
	path := testPipePath()
	ln, err := listenUserPipe(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := New(t.TempDir(), db)
	svc.Attach(engine.NewScheduler(nil), &testSwitch{})
	go svc.Serve(ln)
	// Connections abandoned before sending JSON must not terminate the listener.
	for i := 0; i < 15; i++ {
		c, err := dialPipe(path)
		if err != nil {
			t.Fatal(err)
		}
		c.Close()
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, err := dialPipe(path)
			if err != nil {
				t.Errorf("dial: %v", err)
				return
			}
			req := Request{Op: "status"}
			if i%4 == 0 {
				req = Request{Op: "putRule", Rule: rules.Rule{ID: fmt.Sprint(i), Exe: fmt.Sprintf("app%d.exe", i), Mode: rules.ModeAll, Fallback: rules.FallbackWait}}
			}
			rep, err := Call(c, req)
			if err != nil || !rep.OK {
				t.Errorf("RPC: %+v, %v", rep, err)
			}
			if rep.Status != nil && rep.Status.Running {
				t.Error("saving an app started routing")
			}
		}(i)
	}
	wg.Wait()
	if got := len(svc.ruleList()); got != 8 {
		t.Fatalf("lost concurrent rules: %d", got)
	}
	if svc.isOn() {
		t.Fatal("routing must stay stopped")
	}
	// Paused-engine commands keep working after the polling burst.
	c, err := dialPipe(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Call(c, Request{Op: "deleteRule", ID: "0"}); err != nil {
		t.Fatal(err)
	}
	if got := len(svc.ruleList()); got != 7 {
		t.Fatalf("delete while stopped: %d", got)
	}
}

func TestWindowsPipeCloseUnblocksAccept(t *testing.T) {
	ln, err := listenUserPipe(testPipePath())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		c, err := ln.Accept()
		if c != nil {
			c.Close()
		}
		done <- err
	}()
	ln.Close()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("closed listener accepted a connection")
		}
	case <-time.After(time.Second):
		t.Fatal("closing listener left Accept blocked")
	}
}

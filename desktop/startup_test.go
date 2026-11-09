package main

import (
	"encoding/json"
	"errors"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"net-merge/internal/control"
)

func statusReply(t *testing.T, a *API) control.Reply {
	t.Helper()
	w := httptest.NewRecorder()
	a.ServeHTTP(w, httptest.NewRequest("GET", "/api/status", nil))
	var reply control.Reply
	if err := json.Unmarshal(w.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Status == nil {
		t.Fatalf("missing status: %s", w.Body.String())
	}
	return reply
}

func TestStartingStatusDoesNotReportTransientMissingPipe(t *testing.T) {
	a := &API{stateDir: t.TempDir(), connect: func() (net.Conn, error) { return nil, absentPipeError() }, engineAlive: func() bool { return true }, starting: true, startupDeadline: time.Now().Add(time.Minute)}
	reply := statusReply(t, a)
	if !reply.Status.Starting || reply.Status.Running || reply.Status.Error != "" {
		t.Fatalf("startup shown as an error: %+v", reply.Status)
	}
}

func replyConnection(st control.Status) (net.Conn, error) {
	a, b := net.Pipe()
	go func() {
		defer b.Close()
		var req control.Request
		if json.NewDecoder(b).Decode(&req) == nil {
			_ = json.NewEncoder(b).Encode(control.Reply{OK: true, Status: &st})
		}
	}()
	return a, nil
}

func TestStartupWaitsForRoutingNotJustPipe(t *testing.T) {
	a := &API{starting: true, startupDeadline: time.Now().Add(time.Minute), connect: func() (net.Conn, error) { return replyConnection(control.Status{}) }}
	if got := statusReply(t, a).Status; !got.Starting || got.Error != "" {
		t.Fatalf("not ready yet: %+v", got)
	}
	a.connect = func() (net.Conn, error) { return replyConnection(control.Status{Running: true}) }
	if got := statusReply(t, a).Status; got.Starting || !got.Running || got.Error != "" {
		t.Fatalf("ready: %+v", got)
	}
}

func TestStartupTimeoutRemainsVisible(t *testing.T) {
	a := &API{stateDir: t.TempDir(), starting: true, startupDeadline: time.Now().Add(-time.Second), connect: func() (net.Conn, error) { return nil, absentPipeError() }, engineAlive: func() bool { return false }}
	for i := 0; i < 2; i++ {
		if got := statusReply(t, a).Status; got.Starting || got.Error == "" {
			t.Fatalf("timeout lost: %+v", got)
		}
	}
}

func TestStartLaunchesOnceAndWaitsForEngine(t *testing.T) {
	launches, live := 0, false
	a := &API{stateDir: t.TempDir(), connect: func() (net.Conn, error) { return nil, absentPipeError() }, engineAlive: func() bool { return live }}
	a.launch = func() error { launches++; live = true; return nil }
	if err := a.startRouting(); err != nil {
		t.Fatal(err)
	}
	if got := statusReply(t, a).Status; !got.Starting || got.Error != "" {
		t.Fatalf("launch not pending: %+v", got)
	}
	if err := a.startRouting(); err != nil || launches != 1 {
		t.Fatalf("duplicate launch: %d, %v", launches, err)
	}
	a.connect = func() (net.Conn, error) { return replyConnection(control.Status{Running: true}) }
	if got := statusReply(t, a).Status; !got.Running || got.Starting {
		t.Fatalf("ready: %+v", got)
	}
}

func TestUACDenialRemainsVisible(t *testing.T) {
	denied := errors.New("UAC request was cancelled")
	a := &API{stateDir: t.TempDir(), connect: func() (net.Conn, error) { return nil, absentPipeError() }, engineAlive: func() bool { return false }, launch: func() error { return denied }}
	if err := a.startRouting(); !errors.Is(err, denied) {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if got := statusReply(t, a).Status; got.Starting || got.Error != denied.Error() {
			t.Fatalf("denial hidden: %+v", got)
		}
	}
}

func TestExistingInitializingEngineIsNotRelaunched(t *testing.T) {
	a := &API{stateDir: t.TempDir(), connect: func() (net.Conn, error) { return nil, absentPipeError() }, engineAlive: func() bool { return true }, launch: func() error { t.Fatal("must not launch another engine"); return nil }}
	if err := a.startRouting(); err != nil {
		t.Fatal(err)
	}
	if got := statusReply(t, a).Status; !got.Starting || got.Error != "" {
		t.Fatalf("initializing: %+v", got)
	}
}

func TestStartupDoesNotHideEngineFailure(t *testing.T) {
	a := &API{starting: true, connect: func() (net.Conn, error) { return replyConnection(control.Status{Error: "TUN initialization failed"}) }}
	if got := statusReply(t, a).Status; got.Starting || got.Error != "TUN initialization failed" {
		t.Fatalf("real failure hidden: %+v", got)
	}
}

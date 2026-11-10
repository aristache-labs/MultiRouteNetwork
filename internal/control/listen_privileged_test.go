//go:build !windows

package control

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/rules"
	"MultiRouteNetwork/internal/state"
	"MultiRouteNetwork/internal/store"
)

func TestPrivilegedSocketRejectsTheWrongUserWithoutApplyingTheRequest(t *testing.T) {
	dir := shortServiceDir(t)
	if err := WriteAuth(dir, Auth{UID: 1000, Exe: "/usr/bin/MultiRoute"}); err != nil {
		t.Fatal(err)
	}
	restorePeer(t, func(net.Conn) (int, string, error) { return 1001, "/usr/bin/MultiRoute", nil })
	ln := serveTest(t, dir)
	conn, err := DialPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Call(conn, Request{Op: "putRule", Rule: rules.Rule{Exe: "curl", Mode: rules.ModeAll, Fallback: rules.FallbackWait}})
	if !IsAuthorization(err) {
		t.Fatal(err)
	}
	if svc := ln.svc; len(svc.ruleList()) != 0 {
		t.Fatal("rejected peer changed rules")
	}
}

func TestPrivilegedSocketSurvivesADroppedClient(t *testing.T) {
	dir := shortServiceDir(t)
	if err := WriteAuth(dir, Auth{UID: 1000, Exe: "/usr/bin/MultiRoute"}); err != nil {
		t.Fatal(err)
	}
	restorePeer(t, func(net.Conn) (int, string, error) { return 1000, "/usr/bin/MultiRoute", nil })
	serveTest(t, dir)
	for i := 0; i < 8; i++ {
		conn, err := DialPath(dir)
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}
	conn, err := DialPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Call(conn, Request{Op: "status"})
	if err != nil || !rep.OK || rep.Status == nil || rep.Status.Running {
		t.Fatalf("%+v %v", rep, err)
	}
	if got := ServiceKind(false, dialMissing(t)); got != ServiceUninstalled {
		t.Fatal(got)
	}
	if got := ServiceKind(true, dialMissing(t)); got != ServiceStopped {
		t.Fatal(got)
	}
}

func TestPrivilegedSocketRejectsASymlink(t *testing.T) {
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "service")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ListenPrivileged(link); !errors.Is(err, state.ErrSymlink) {
		t.Fatal(err)
	}
}

type testListen struct{ svc *Service }

func serveTest(t *testing.T, dir string) testListen {
	t.Helper()
	ln, err := ListenPrivileged(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	svc := New(dir, db)
	svc.Attach(engine.NewScheduler(nil), &testSwitch{})
	go svc.Serve(ln)
	return testListen{svc: svc}
}

func restorePeer(t *testing.T, fn func(net.Conn) (int, string, error)) {
	t.Helper()
	prev := peerOf
	peerOf = fn
	t.Cleanup(func() { peerOf = prev })
}

// macOS rejects a unix socket path longer than 104 bytes. t.TempDir includes the test name.
// /tmp is a symlink there, and the service refuses a symlinked parent.
func shortServiceDir(t *testing.T) string {
	t.Helper()
	parent := "/tmp"
	if fi, err := os.Lstat(parent); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		parent = "/private/tmp"
	}
	dir, err := os.MkdirTemp(parent, "mr")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func dialMissing(t *testing.T) error {
	t.Helper()
	_, err := DialPath(filepath.Join(t.TempDir(), "absent"))
	if err == nil {
		t.Fatal("dialed a missing socket")
	}
	return err
}

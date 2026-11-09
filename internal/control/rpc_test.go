package control

import (
	"net"
	"path/filepath"
	"testing"

	"net-merge/internal/engine"
	"net-merge/internal/nics"
	"net-merge/internal/rules"
	"net-merge/internal/store"
)

func TestStatusKeepsConnectedAdapters(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "s"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	svc := New(t.TempDir(), db)
	sw := &testSwitch{}
	sw.On()
	svc.Attach(engine.NewScheduler(nil), sw)
	svc.Tick([]engine.SlotStats{{
		Name: "Wi-Fi", Present: true, IP: "172.31.21.40", State: "up", LinkIn: 1000,
	}}, []nics.Skip{{Name: "Tailscale", Reason: "virtual/VPN adapter (name)"}})
	st := svc.Snapshot(nil, nil, true)
	if len(st.Adapters) != 2 || st.Adapters[0].Name != "Wi-Fi" || !st.Adapters[0].Routable || st.Adapters[1].State != "VPN" {
		t.Fatalf("%+v", st.Adapters)
	}
	if st.Adapters[0].TodayDown != 1000 || st.Adapters[0].MonthDown != 1000 {
		t.Fatalf("totals %+v", st.Adapters[0])
	}
}

func TestOwnTunnelIsHidden(t *testing.T) {
	if !hideAdapter("net-merge", "looks like a virtual/VPN adapter (name)") {
		t.Fatal("the engine tunnel should not be shown as an adapter")
	}
}

func TestSnapshotListsOnlyAddedApps(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "s"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	svc := New(t.TempDir(), db)
	if err := svc.PutRules([]rules.Rule{{ID: "1", Exe: "chrome.exe", Mode: rules.ModeAll, Fallback: rules.FallbackWait}}); err != nil {
		t.Fatal(err)
	}
	svc.account("lsass.exe", `C:\Windows\System32\lsass.exe`, "Wi-Fi", rules.Rule{}, true, 100, false)
	st := svc.Snapshot(nil, nil, false)
	if len(st.Apps) != 1 || st.Apps[0].Exe != "chrome.exe" || st.Apps[0].DownBytes != 0 {
		t.Fatalf("%+v", st.Apps)
	}
}

func TestPipeRoundTrip(t *testing.T) {
	a, b := net.Pipe()
	db, err := store.Open(filepath.Join(t.TempDir(), "s"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(t.TempDir(), db)
	t.Cleanup(func() { db.Close() })
	go svc.serveConn(a)
	rep, err := Call(b, Request{Op: "putRule", Rule: rules.Rule{Exe: "curl.exe", Mode: rules.ModeAll, Fallback: rules.FallbackWait}})
	if err != nil || !rep.OK {
		t.Fatal(err, rep.Error)
	}
	rep, err = Call(mustDial(t, svc), Request{Op: "status"})
	if err != nil || rep.Status == nil || len(rep.Status.Rules) != 1 {
		t.Fatalf("%v %+v", err, rep)
	}
}

func mustDial(t *testing.T, svc *Service) net.Conn {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go svc.Serve(ln)
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

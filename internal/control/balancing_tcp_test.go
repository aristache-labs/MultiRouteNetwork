package control

import (
	"context"
	"net"
	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/nics"
	"MultiRouteNetwork/internal/rules"
	"testing"
	"time"
)

func TestMultipleTCPConnectionsUseAdaptersTogether(t *testing.T) {
	for _, mode := range []rules.Mode{rules.ModeAll, rules.ModeSeveral} {
		t.Run(string(mode), func(t *testing.T) {
			ln, err := net.Listen("tcp4", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			accepted := make(chan net.Conn)
			done := make(chan struct{})
			defer close(done)
			go func() {
				for {
					c, e := ln.Accept()
					if e != nil {
						return
					}
					select {
					case accepted <- c:
					case <-done:
						c.Close()
						return
					}
				}
			}()
			ns := []nics.Nic{{Name: "A", IP: net.ParseIP("127.0.0.1"), Weight: 2}, {Name: "B", IP: net.ParseIP("127.0.0.2"), Weight: 1}}
			if mode == rules.ModeSeveral {
				ns = append(ns, nics.Nic{Name: "C", IP: net.ParseIP("127.0.0.3"), Weight: 100})
			}
			sched := engine.NewScheduler(ns)
			sched.SetAuto(true)
			svc := New("", nil)
			svc.PutRules([]rules.Rule{{Exe: "app.exe", Mode: mode, Adapters: []string{"A", "B"}, Preferred: "A", Fallback: rules.FallbackWait}})
			svc.Attach(sched, &testSwitch{})
			bal := engine.NewBalancer(sched)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ctx = engine.WithHint(ctx, engine.FlowHint{Name: "app.exe", Known: true})
			counts := map[string]int{}
			clients := []net.Conn{}
			defer func() {
				for _, c := range clients {
					c.Close()
				}
			}()
			for i := 0; i < 12; i++ {
				c, e := bal.DialPinned(ctx, "127.0.0.1", ln.Addr().(*net.TCPAddr).Port)
				if e != nil {
					t.Fatal(e)
				}
				clients = append(clients, c)
				counts[c.LocalAddr().(*net.TCPAddr).IP.String()]++
				select {
				case server := <-accepted:
					defer server.Close()
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			if counts["127.0.0.1"] != 8 || counts["127.0.0.2"] != 4 || len(counts) != 2 {
				t.Fatalf("simultaneous connections: %v", counts)
			}
			svc.PutRules([]rules.Rule{{Exe: "app.exe", Mode: rules.ModeOne, Adapters: []string{"B"}, Fallback: rules.FallbackWait}})
			c, e := bal.DialPinned(ctx, "127.0.0.1", ln.Addr().(*net.TCPAddr).Port)
			if e != nil {
				t.Fatal(e)
			}
			defer c.Close()
			if c.LocalAddr().(*net.TCPAddr).IP.String() != "127.0.0.2" {
				t.Fatal("one adapter rule ignored")
			}
			select {
			case server := <-accepted:
				defer server.Close()
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}

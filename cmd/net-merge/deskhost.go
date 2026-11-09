package main

import (
	"net-merge/internal/control"
	"net-merge/internal/engine"
	"net-merge/internal/logx"
	"net-merge/internal/nics"
	"net-merge/internal/store"
	"time"
)

func hostControl(dir string, sched *engine.Scheduler, bal *engine.Balancer, sw control.Switch) *control.Service {
	db, err := store.Open(dir)
	if err != nil {
		logx.Printf("warning: settings store: %v\n", err)
		return nil
	}
	svc := control.New(dir, db)
	svc.EnableCurlDiagnostics()
	svc.Attach(sched, sw)
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		last := time.Now()
		var scanned time.Time
		var skipped []nics.Skip
		for now := range ticker.C {
			if now.Sub(scanned) >= 3*time.Second {
				_, skipped, _ = nics.DiscoverNicsDetailed(nil, nil)
				scanned = now
			}
			svc.Sample(sched.Snapshot(), skipped, now.Sub(last))
			last = now
		}
	}()
	if bal != nil {
		bal.Prepare = svc.Prepare
	}
	ln, err := control.Listen()
	if err != nil {
		logx.Printf("warning: desktop pipe: %v\n", err)
		return svc
	}
	go svc.Serve(ln)
	return svc
}

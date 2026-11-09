package control

import (
	"errors"
	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/nics"
	"MultiRouteNetwork/internal/store"
	"time"
)

// LocalRules saves configuration without starting routing. The desktop must only
// call this after confirming that the engine and its pipe are both absent.
func LocalRules(dir string, req Request) (Reply, error) {
	if req.Op != "putRule" && req.Op != "deleteRule" && req.Op != "putSettings" {
		return Reply{}, errors.New("unsupported offline operation")
	}
	db, err := store.Open(dir)
	if err != nil {
		return Reply{}, err
	}
	defer db.Close()
	svc := New(dir, db)
	rep := svc.handle(req)
	if !rep.OK {
		return rep, errors.New(rep.Error)
	}
	return rep, nil
}

// LocalHistory keeps the usage screen readable while the engine is not running.
func LocalHistory(dir string, req Request) (Reply, error) {
	db, err := store.OpenReadOnly(dir)
	if err != nil {
		if req.Op == "csv" {
			return Reply{OK: true, CSV: "exe,path,down_bytes,up_bytes,total_bytes\n"}, nil
		}
		return Reply{OK: true, Usage: []store.UsageRow{}}, nil
	}
	defer db.Close()
	svc := New(dir, db)
	if req.Op == "csv" {
		csv, err := svc.UsageCSV(rangeStart(req.Range, time.Now()))
		return Reply{OK: err == nil, CSV: csv}, err
	}
	rows, err := svc.Usage(rangeStart(req.Range, time.Now()))
	return Reply{OK: err == nil, Usage: rows}, err
}

// LocalStatus discovers inventory without starting the privileged routing engine.
// The database is read-only so the desktop cannot migrate or overwrite engine state.
func LocalStatus(dir string) Status {
	db, _ := store.OpenReadOnly(dir)
	if db != nil {
		defer db.Close()
	}
	svc := New(dir, db)
	adapters, skipped, err := nics.DiscoverNicsDetailed(nil, nil)
	if err != nil {
		svc.SetError(err.Error())
	}
	snap := make([]engine.SlotStats, 0, len(adapters))
	for _, a := range adapters {
		sl := engine.SlotStats{Name: a.Name, Present: true, State: "up"}
		if a.IP != nil {
			sl.IP = a.IP.String()
		}
		if a.IP6 != nil {
			sl.IP6 = a.IP6.String()
		}
		snap = append(snap, sl)
	}
	return svc.Snapshot(snap, skipped, false)
}

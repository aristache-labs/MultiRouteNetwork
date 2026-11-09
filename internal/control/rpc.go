package control

import (
	"encoding/json"
	"errors"
	"net"
	"time"

	"MultiRouteNetwork/internal/rules"
	"MultiRouteNetwork/internal/store"
)

// Request is one desktop call. Op selects the fields that matter.
type Request struct {
	Op       string         `json:"op"`
	Rule     rules.Rule     `json:"rule,omitempty"`
	ID       string         `json:"id,omitempty"`
	Settings store.Settings `json:"settings,omitempty"`
	Range    string         `json:"range,omitempty"`
	Running  bool           `json:"running,omitempty"`
}

// Reply is the JSON answer. Status, usage or csv is set depending on Op.
type Reply struct {
	OK     bool             `json:"ok"`
	Error  string           `json:"error,omitempty"`
	Status *Status          `json:"status,omitempty"`
	Usage  []store.UsageRow `json:"usage,omitempty"`
	CSV    string           `json:"csv,omitempty"`
}

func (s *Service) Serve(ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go s.serveConn(c)
	}
}

func (s *Service) serveConn(c net.Conn) {
	defer c.Close()
	dec := json.NewDecoder(c)
	enc := json.NewEncoder(c)
	for {
		_ = c.SetDeadline(time.Now().Add(10 * time.Second))
		var req Request
		if err := dec.Decode(&req); err != nil {
			return
		}
		_ = enc.Encode(s.handle(req))
	}
}

func (s *Service) handle(req Request) Reply {
	switch req.Op {
	case "status":
		st := s.Snapshot(nil, nil, s.isOn())
		return Reply{OK: true, Status: &st}
	case "start":
		if err := s.Start(); err != nil {
			return Reply{Error: err.Error()}
		}
		return Reply{OK: true}
	case "stop":
		s.Stop()
		return Reply{OK: true}
	case "quit":
		go func() {
			time.Sleep(200 * time.Millisecond)
			s.Quit()
		}()
		return Reply{OK: true}
	case "putRule":
		return s.putRule(req.Rule)
	case "deleteRule":
		return s.deleteRule(req.ID)
	case "putSettings":
		if err := s.PutSettings(req.Settings); err != nil {
			return Reply{Error: err.Error()}
		}
		return Reply{OK: true}
	case "usage":
		rows, err := s.Usage(rangeStart(req.Range, time.Now()))
		if err != nil {
			return Reply{Error: err.Error()}
		}
		return Reply{OK: true, Usage: rows}
	case "csv":
		text, err := s.UsageCSV(rangeStart(req.Range, time.Now()))
		if err != nil {
			return Reply{Error: err.Error()}
		}
		return Reply{OK: true, CSV: text}
	case "snapshot":
		st := s.Snapshot(nil, nil, req.Running)
		return Reply{OK: true, Status: &st}
	default:
		return Reply{Error: "unknown op"}
	}
}

func (s *Service) isOn() bool {
	s.mu.Lock()
	sw := s.sw
	s.mu.Unlock()
	return sw != nil && sw.IsOn()
}

func (s *Service) putRule(r rules.Rule) Reply {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	if r.Exe == "" && r.Path == "" {
		return Reply{Error: "a rule needs an executable name or path"}
	}
	if r.ID == "" {
		r.ID = time.Now().Format("20060102T150405.000")
	}
	if r.Fallback == "" {
		r.Fallback = rules.FallbackWait
	}
	if r.Mode == "" {
		r.Mode = rules.ModeAll
	}
	list := append([]rules.Rule{}, s.ruleList()...)
	replaced := false
	for i := range list {
		if list[i].ID == r.ID || (r.ExactPath && list[i].ExactPath && list[i].Path == r.Path) || (!r.ExactPath && !list[i].ExactPath && list[i].Exe == r.Exe) {
			list[i] = r
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, r)
	}
	if err := s.PutRules(list); err != nil {
		return Reply{Error: err.Error()}
	}
	return Reply{OK: true}
}

func (s *Service) deleteRule(id string) Reply {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	var list []rules.Rule
	for _, r := range s.ruleList() {
		if r.ID != id {
			list = append(list, r)
		}
	}
	if err := s.PutRules(list); err != nil {
		return Reply{Error: err.Error()}
	}
	return Reply{OK: true}
}

func rangeStart(name string, now time.Time) time.Time {
	local := now.In(time.Local)
	switch name {
	case "7d":
		return local.AddDate(0, 0, -7)
	case "30d":
		return local.AddDate(0, 0, -30)
	case "month":
		return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.Local)
	case "all":
		return time.Unix(0, 0)
	default:
		return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
	}
}

// Call performs one request against a connection.
func Call(conn net.Conn, req Request) (Reply, error) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Reply{}, err
	}
	var rep Reply
	if err := json.NewDecoder(conn).Decode(&rep); err != nil {
		return Reply{}, err
	}
	if !rep.OK && rep.Error != "" {
		return rep, errors.New(rep.Error)
	}
	return rep, nil
}

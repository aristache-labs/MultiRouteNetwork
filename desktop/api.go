package main

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"MultiRouteNetwork/internal/control"
	"MultiRouteNetwork/internal/platform"
	"MultiRouteNetwork/internal/rules"
	"MultiRouteNetwork/internal/state"
	"MultiRouteNetwork/internal/store"
)

// API is the local HTTP surface the window uses. It stays unelevated and talks to the engine pipe.
type API struct {
	configureStartup func(bool) error

	quitting        atomic.Bool
	window          *windowControl
	mu              sync.Mutex
	stateDir        string
	connect         func() (net.Conn, error)
	engineAlive     func() bool
	launch          func() error
	startupMu       sync.Mutex
	starting        bool
	startupDeadline time.Time
	startupError    string
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(r.URL.Path, "/api")
	if path == "/icon" {
		serveAppIcon(w, r)
		return
	}
	var rep control.Reply
	var err error
	switch path {
	case "/window/preferences":
		if a.window == nil {
			err = errors.New("window controls are unavailable")
			break
		}
		if r.Method == http.MethodPost {
			var body struct {
				Action string `json:"action"`
			}
			err = json.NewDecoder(r.Body).Decode(&body)
			if err == nil {
				err = a.window.savePreference(body.Action)
			}
			if err != nil {
				break
			}
		}
		_ = json.NewEncoder(w).Encode(a.window.preferences())
		return
	case "/window/close":
		if a.window == nil {
			err = errors.New("window controls are unavailable")
			break
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Action   string `json:"action"`
			Remember bool   `json:"remember"`
		}
		err = json.NewDecoder(r.Body).Decode(&body)
		if err == nil {
			err = a.window.choose(body.Action, body.Remember)
		}
		rep = control.Reply{OK: err == nil}
	case "/status":
		rep, err = a.call(control.Request{Op: "status"})
		pipeErr := err
		if err != nil {
			st := control.LocalStatus(a.dir())
			rep = control.Reply{OK: true, Status: &st}
			err = nil
		}
		a.decorateStatus(rep.Status, pipeErr)
	case "/start":
		err = a.startRouting()
		rep = control.Reply{OK: err == nil}
	case "/stop":
		rep, err = a.call(control.Request{Op: "stop"})
	case "/rule":
		var rule rules.Rule
		err = json.NewDecoder(r.Body).Decode(&rule)
		if err == nil {
			rep, err = a.call(control.Request{Op: "putRule", Rule: rule})
		}
	case "/rule/delete":
		var body struct{ ID string }
		err = json.NewDecoder(r.Body).Decode(&body)
		if err == nil {
			rep, err = a.call(control.Request{Op: "deleteRule", ID: body.ID})
		}
	case "/settings":
		var st store.Settings
		err = json.NewDecoder(r.Body).Decode(&st)
		if err == nil {
			previous := control.LocalStatus(a.dir()).Settings
			if previous.StartWindows != st.StartWindows {
				err = a.updateAutostart(st.StartWindows)
			}
			if err == nil {
				rep, err = a.call(control.Request{Op: "putSettings", Settings: st})
			}
			if err != nil && previous.StartWindows != st.StartWindows {
				_ = a.updateAutostart(previous.StartWindows)
			}
		}
	case "/usage":
		rep, err = a.call(control.Request{Op: "usage", Range: r.URL.Query().Get("range")})
	case "/csv":
		rep, err = a.call(control.Request{Op: "csv", Range: r.URL.Query().Get("range")})
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(control.Reply{Error: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(rep)
}

func (a *API) call(req control.Request) (control.Reply, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	dial := a.connect
	if dial == nil {
		dial = control.Dial
	}
	conn, err := dial()
	if err != nil {
		if control.IsPipeAbsent(err) && !a.runningEngine() {
			if req.Op == "putRule" || req.Op == "deleteRule" || req.Op == "putSettings" {
				return control.LocalRules(a.dir(), req)
			}
		}
		if req.Op == "usage" || req.Op == "csv" {
			return control.LocalHistory(a.dir(), req)
		}
		return control.Reply{}, err
	}
	return control.Call(conn, req)
}

func (a *API) dir() string {
	if a.stateDir != "" {
		return a.stateDir
	}
	return state.DefaultDir()
}

func (a *API) runningEngine() bool {
	if a.engineAlive != nil {
		return a.engineAlive()
	}
	return platform.InstanceRunning()
}

func enginePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "MultiRouteNetwork.exe"
	}
	next := filepath.Join(filepath.Dir(exe), "MultiRouteNetwork.exe")
	if _, err := os.Stat(next); err == nil {
		return next
	}
	return next
}

var errNoEngine = errors.New("MultiRouteNetwork.exe was not found next to MultiRoute")

func (a *API) updateAutostart(enabled bool) error {
	if a.configureStartup != nil {
		return a.configureStartup(enabled)
	}
	return setDesktopAutostart(enabled)
}

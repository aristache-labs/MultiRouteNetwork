package main

import (
	"MultiRouteNetwork/internal/control"
	"MultiRouteNetwork/internal/icon"
	"fmt"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

type trayView struct {
	header, tooltip, toggle string
	rows                    []string
	running, starting       bool
}

func trayRate(bps float64) string {
	mbps := bps * 8 / 1e6
	if mbps >= 1 {
		return fmt.Sprintf("%.1f Mbit/s", mbps)
	}
	return fmt.Sprintf("%.0f kbit/s", mbps*1000)
}
func desktopTrayView(st *control.Status) trayView {
	if st == nil {
		st = &control.Status{}
	}
	state, toggle := "Offline", "Start"
	switch st.Service {
	case control.ServiceUninstalled:
		state = "Service not installed"
	case control.ServiceAuthorization:
		state = "Authorization required"
	case control.ServiceStopped:
		state = "Engine stopped"
	case control.ServiceUnreachable:
		state = "Communication error"
	default:
		if st.Running {
			state, toggle = "Online", "Stop"
		}
		if st.Starting {
			state, toggle = "Starting…", "Starting…"
		}
	}
	v := trayView{header: fmt.Sprintf("%s · Total ↓ %s ↑ %s", state, trayRate(st.DownBps), trayRate(st.UpBps)), toggle: toggle, running: st.Running, starting: st.Starting}
	adapters := append([]control.AdapterCard(nil), st.Adapters...)
	sort.SliceStable(adapters, func(i, j int) bool { return adapters[i].Name < adapters[j].Name })
	for _, a := range adapters {
		suffix := ""
		if a.Excluded {
			suffix = " (not routed)"
		}
		v.rows = append(v.rows, fmt.Sprintf("%s: ↓ %s ↑ %s%s", a.Name, trayRate(a.DownBps), trayRate(a.UpBps), suffix))
	}
	v.tooltip = fmt.Sprintf("MultiRoute · %s\n↓ %s ↑ %s", state, trayRate(st.DownBps), trayRate(st.UpBps))
	r := []rune(v.tooltip)
	if len(r) > 63 {
		v.tooltip = string(r[:62]) + "…"
	}
	return v
}
func installDesktopTray(app *application.App, api *API, show func(), quit func() error) {
	menu := app.NewMenu()
	menu.Add("Show MultiRoute").OnClick(func(*application.Context) { show() })
	menu.AddSeparator()
	header := menu.Add("MultiRoute · Offline").SetEnabled(false)
	const maxRows = 16
	rows := make([]*application.MenuItem, maxRows)
	for i := range rows {
		rows[i] = menu.Add("").SetEnabled(false).SetHidden(true)
	}
	extra := menu.Add("").SetEnabled(false).SetHidden(true)
	menu.AddSeparator()
	var online, busy, closed atomic.Bool
	report := func(err error) {
		if err != nil {
			api.window.prompt(err.Error())
		}
	}
	toggle := menu.Add("Start")
	toggle.OnClick(func(*application.Context) {
		if !busy.CompareAndSwap(false, true) {
			return
		}
		go func() {
			defer busy.Store(false)
			if online.Load() {
				rep, err := api.call(control.Request{Op: "stop"})
				if err == nil && !rep.OK {
					err = fmt.Errorf("%s", rep.Error)
				}
				report(err)
			} else {
				report(api.startRouting())
			}
		}()
	})
	menu.Add("Exit").OnClick(func(*application.Context) { go func() { report(quit()) }() })
	tray := app.SystemTray.New()
	iconOn, iconOff := icon.PNG(true), icon.PNG(false)
	tray.SetMenu(menu).SetIcon(iconOff)
	tray.SetTooltip("MultiRoute · Offline")
	tray.OnDoubleClick(show)
	lastIconOn := false
	done := make(chan struct{})
	app.OnShutdown(func() { closed.Store(true); close(done) })
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			last := ""
			for {
				rep, err := api.call(control.Request{Op: "status"})
				st := rep.Status
				if err != nil || !rep.OK || st == nil {
					local := control.LocalStatus(api.dir())
					st = &local
				}
				api.decorateStatus(st, err)
				v := desktopTrayView(st)
				signature := v.header + v.toggle + strings.Join(v.rows, "\n")
				if signature != last {
					last = signature
					application.InvokeAsync(func() {
						if closed.Load() {
							return
						}
						online.Store(v.running)
						header.SetLabel(v.header)
						toggle.SetLabel(v.toggle).SetEnabled(!v.starting)
						for i, row := range rows {
							if i < len(v.rows) {
								row.SetLabel(v.rows[i]).SetHidden(false)
							} else {
								row.SetHidden(true)
							}
						}
						if len(v.rows) > maxRows {
							extra.SetLabel(fmt.Sprintf("%d more adapters — open MultiRoute", len(v.rows)-maxRows)).SetHidden(false)
						} else {
							extra.SetHidden(true)
						}
						tray.SetTooltip(v.tooltip)
						if v.running != lastIconOn {
							if v.running {
								tray.SetIcon(iconOn)
							} else {
								tray.SetIcon(iconOff)
							}
							lastIconOn = v.running
						}
					})
				}
				select {
				case <-done:
					return
				case <-ticker.C:
				}
			}
		}()
	})
}

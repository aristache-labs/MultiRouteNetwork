package main

import (
	"time"

	"MultiRouteNetwork/internal/control"
)

const engineStartupTimeout = 30 * time.Second

func (a *API) startRouting() error {
	a.startupMu.Lock()
	if a.starting {
		a.startupMu.Unlock()
		return nil
	}
	a.starting, a.startupError, a.startupDeadline = true, "", time.Time{}
	a.startupMu.Unlock()

	_, err := a.call(control.Request{Op: "start"})
	if err == nil {
		a.finishStartup(nil)
		return nil
	}
	if !control.IsPipeAbsent(err) {
		a.finishStartup(err)
		return err
	}
	// A process already holding the engine mutex may still be initializing its
	// pipe. Wait for it instead of spawning a competing elevated process.
	if !a.runningEngine() {
		launch := a.launch
		if launch == nil {
			launch = launchEngine
		}
		if err := launch(); err != nil {
			a.finishStartup(err)
			return err
		}
	}
	// ShellExecute returns after UAC. Time spent answering that prompt must not
	// consume the engine's readiness window.
	a.startupMu.Lock()
	if a.starting {
		a.startupDeadline = time.Now().Add(engineStartupTimeout)
	}
	a.startupMu.Unlock()
	return nil
}

func (a *API) finishStartup(err error) {
	a.startupMu.Lock()
	defer a.startupMu.Unlock()
	a.starting, a.startupDeadline, a.startupError = false, time.Time{}, ""
	if err != nil {
		a.startupError = err.Error()
	}
}

func (a *API) decorateStatus(st *control.Status, pipeErr error) {
	if st == nil {
		return
	}
	a.startupMu.Lock()
	defer a.startupMu.Unlock()
	if st.Running {
		a.starting, a.startupError, a.startupDeadline = false, "", time.Time{}
		return
	}
	if a.starting {
		if pipeErr == nil && st.Error != "" {
			a.starting, a.startupError = false, st.Error
		} else if !a.startupDeadline.IsZero() && !time.Now().Before(a.startupDeadline) {
			a.starting = false
			a.startupError = "The routing engine did not become ready within 30 seconds. Try Start again."
		} else {
			st.Starting, st.Error = true, ""
			return
		}
	}
	if a.startupError != "" {
		st.Error = a.startupError
		return
	}
	if pipeErr != nil && (!control.IsPipeAbsent(pipeErr) || a.runningEngine()) {
		st.Error = "Cannot communicate with the routing engine. Close MultiRoute with Exit and reopen it."
	}
}

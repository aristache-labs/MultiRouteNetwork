package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type windowPreference struct {
	Action string `json:"action"`
}

type windowControl struct {
	mu         sync.Mutex
	dir        string
	pending    bool
	closeError string
	hide       func()
	exit       func() error
	prompt     func(string)
}

func validCloseAction(action string) bool {
	return action == "ask" || action == "tray" || action == "exit"
}

func (c *windowControl) loadPreferenceLocked() string {
	data, err := os.ReadFile(filepath.Join(c.dir, "window-preferences.json"))
	var pref windowPreference
	if err != nil || json.Unmarshal(data, &pref) != nil || !validCloseAction(pref.Action) {
		return "ask"
	}
	return pref.Action
}

func (c *windowControl) savePreferenceLocked(action string) error {
	if !validCloseAction(action) {
		return errors.New("invalid close action")
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	data, _ := json.Marshal(windowPreference{Action: action})
	f, err := os.CreateTemp(c.dir, ".window-preferences-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(c.dir, "window-preferences.json"))
}

func (c *windowControl) savePreference(action string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.savePreferenceLocked(action)
}

func (c *windowControl) preferences() any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return struct {
		OK         bool   `json:"ok"`
		Action     string `json:"action"`
		Pending    bool   `json:"pending"`
		CloseError string `json:"closeError,omitempty"`
	}{true, c.loadPreferenceLocked(), c.pending, c.closeError}
}

func (c *windowControl) requestClose() {
	c.mu.Lock()
	if c.pending {
		c.mu.Unlock()
		return
	}
	action := c.loadPreferenceLocked()
	c.pending, c.closeError = true, ""
	c.mu.Unlock()
	if action == "ask" {
		c.prompt("")
		return
	}
	if err := c.perform(action); err != nil {
		c.mu.Lock()
		c.closeError = err.Error()
		c.mu.Unlock()
		c.prompt(err.Error())
		return
	}
	c.mu.Lock()
	c.pending = false
	c.mu.Unlock()
}

func (c *windowControl) perform(action string) error {
	if action == "tray" {
		if c.hide != nil {
			c.hide()
		}
		return nil
	}
	if action == "exit" && c.exit != nil {
		return c.exit()
	}
	return errors.New("close action is unavailable")
}

func (c *windowControl) choose(action string, remember bool) error {
	c.mu.Lock()
	if action == "cancel" {
		c.pending, c.closeError = false, ""
		c.mu.Unlock()
		return nil
	}
	if action != "tray" && action != "exit" {
		c.mu.Unlock()
		return errors.New("choose tray or exit")
	}
	preference := "ask"
	if remember {
		preference = action
	}
	if err := c.savePreferenceLocked(preference); err != nil {
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()
	if err := c.perform(action); err != nil {
		return err
	}
	c.mu.Lock()
	c.pending, c.closeError = false, ""
	c.mu.Unlock()
	return nil
}

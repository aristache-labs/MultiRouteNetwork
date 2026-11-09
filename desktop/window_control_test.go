package main

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowCloseChoiceIsRememberedAcrossSessions(t *testing.T) {
	for _, action := range []string{"tray", "exit"} {
		t.Run(action, func(t *testing.T) {
			dir := t.TempDir()
			prompts, hides, exits := 0, 0, 0
			controller := func() *windowControl {
				return &windowControl{dir: dir, prompt: func(string) { prompts++ }, hide: func() { hides++ }, exit: func() error { exits++; return nil }}
			}
			c := controller()
			c.requestClose()
			c.requestClose() // Repeated X must not open another modal.
			if prompts != 1 || hides != 0 || exits != 0 {
				t.Fatal("default close did not ask exactly once")
			}
			if err := c.choose(action, true); err != nil {
				t.Fatal(err)
			}
			c = controller() // A new desktop session reads the saved choice.
			c.requestClose()
			if prompts != 1 {
				t.Fatal("remembered action showed the question again")
			}
			if action == "tray" && (hides != 2 || exits != 0) {
				t.Fatalf("hide=%d exit=%d", hides, exits)
			}
			if action == "exit" && (exits != 2 || hides != 0) {
				t.Fatalf("hide=%d exit=%d", hides, exits)
			}
			if err := c.savePreference("ask"); err != nil {
				t.Fatal(err)
			}
			c.requestClose()
			if prompts != 2 {
				t.Fatal("Ask every time did not restore the question")
			}
		})
	}
}

func TestUncheckedRememberAndCancelKeepAsking(t *testing.T) {
	prompts, hides := 0, 0
	c := &windowControl{dir: t.TempDir(), prompt: func(string) { prompts++ }, hide: func() { hides++ }}
	c.requestClose()
	if err := c.choose("cancel", true); err != nil {
		t.Fatal(err)
	}
	c.requestClose()
	if prompts != 2 || hides != 0 {
		t.Fatal("Cancel performed or remembered an action")
	}
	if err := c.choose("tray", false); err != nil {
		t.Fatal(err)
	}
	c.requestClose()
	if prompts != 3 || hides != 1 {
		t.Fatal("unchecked choice was remembered")
	}
}

func TestInvalidPreferenceAsksAndFailedExitDoesNotHide(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "window-preferences.json"), []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	prompts, hides := 0, 0
	c := &windowControl{dir: dir, prompt: func(string) { prompts++ }, hide: func() { hides++ }, exit: func() error { return errors.New("engine could not stop") }}
	c.requestClose()
	if prompts != 1 {
		t.Fatal("invalid preference should safely ask")
	}
	if err := c.choose("exit", true); err == nil || hides != 0 {
		t.Fatal("failed exit was silently accepted")
	}
	if err := c.choose("cancel", false); err != nil {
		t.Fatal(err)
	}
	c.requestClose()
	if prompts != 2 {
		t.Fatal("failed remembered action must show the question")
	}
	if err := c.savePreference("anything"); err == nil {
		t.Fatal("invalid action accepted")
	}
}

func TestWindowCloseHTTPDoesNotRequireEngine(t *testing.T) {
	hides := 0
	a := &API{window: &windowControl{dir: t.TempDir(), hide: func() { hides++ }}}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, httptest.NewRequest("POST", "/api/window/close", strings.NewReader(`{"action":"tray","remember":true}`)))
	if w.Code != 200 || hides != 1 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	a.ServeHTTP(w, httptest.NewRequest("GET", "/api/window/preferences", nil))
	var got struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.Action != "tray" {
		t.Fatalf("%s %v", w.Body.String(), err)
	}
}

//go:build linux

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func setDesktopAutostart(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "autostart")
	path := filepath.Join(dir, "dev.aristache.multiroute.desktop")
	if !enabled {
		err = os.Remove(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body := "[Desktop Entry]\nType=Application\nName=MultiRoute\nExec=" + quoteDesktop(exe) + " --background\nTerminal=false\nX-GNOME-Autostart-enabled=true\n"
	return os.WriteFile(path, []byte(body), 0o644)
}

func quoteDesktop(path string) string {
	if strings.ContainsAny(path, " \t") {
		return strconvQuote(path)
	}
	return path
}

func strconvQuote(path string) string {
	return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
}

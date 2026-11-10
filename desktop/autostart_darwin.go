//go:build darwin

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func setDesktopAutostart(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	path := filepath.Join(dir, "dev.aristache.multiroute.gui.plist")
	domain := "gui/" + strconv.Itoa(os.Getuid())
	target := domain + "/dev.aristache.multiroute.gui"
	if !enabled {
		_ = exec.Command("launchctl", "bootout", target).Run()
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
	body := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>dev.aristache.multiroute.gui</string>
<key>ProgramArguments</key><array><string>` + xmlEscape(exe) + `</string><string>--background</string></array>
<key>RunAtLoad</key><true/>
</dict></plist>
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "bootout", target).Run()
	return exec.Command("launchctl", "bootstrap", domain, path).Run()
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

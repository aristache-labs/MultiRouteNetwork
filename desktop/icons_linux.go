//go:build linux

package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func executableIcon(path string) ([]byte, error) {
	name, err := desktopIconName(path)
	if err != nil {
		return nil, err
	}
	file, err := resolvePNG(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(file)
}

func desktopIconName(exe string) (string, error) {
	dirs := []string{"/usr/share/applications", "/usr/local/share/applications"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "applications"))
	}
	base := filepath.Base(exe)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".desktop") {
				continue
			}
			execLine, icon, err := readDesktopEntry(filepath.Join(dir, entry.Name()))
			fields := strings.Fields(execLine)
			if err != nil || icon == "" || len(fields) == 0 {
				continue
			}
			token := strings.Trim(fields[0], `"`)
			if token == exe || filepath.Base(token) == base {
				return icon, nil
			}
		}
	}
	return "", os.ErrNotExist
}

func readDesktopEntry(path string) (execLine, icon string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "Exec="); ok {
			execLine = rest
		}
		if rest, ok := strings.CutPrefix(line, "Icon="); ok {
			icon = rest
		}
	}
	return execLine, icon, sc.Err()
}

func resolvePNG(icon string) (string, error) {
	if filepath.IsAbs(icon) && strings.EqualFold(filepath.Ext(icon), ".png") {
		return icon, nil
	}
	name := icon
	if filepath.Ext(name) != "" {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	candidates := []string{
		filepath.Join("/usr/share/icons/hicolor/48x48/apps", name+".png"),
		filepath.Join("/usr/share/icons/hicolor/128x128/apps", name+".png"),
		filepath.Join("/usr/share/pixmaps", name+".png"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".local", "share", "icons", "hicolor", "48x48", "apps", name+".png"))
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}

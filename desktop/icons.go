package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var appIcons = struct {
	sync.Mutex
	images map[string][]byte
}{images: make(map[string][]byte)}

func serveAppIcon(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Query().Get("path"))
	// Only read icons from local executables, never UNC shares, device paths or URLs.
	if !filepath.IsAbs(path) || strings.HasPrefix(path, `\\`) || !strings.EqualFold(filepath.Ext(path), ".exe") {
		http.NotFound(w, r)
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	key := strings.ToLower(path) + info.ModTime().String()
	appIcons.Lock()
	data := appIcons.images[key]
	appIcons.Unlock()
	if data == nil {
		data, err = executableIcon(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		appIcons.Lock()
		if len(appIcons.images) < 256 {
			appIcons.images[key] = data
		}
		appIcons.Unlock()
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(data)
}

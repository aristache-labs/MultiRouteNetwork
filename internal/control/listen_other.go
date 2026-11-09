//go:build !windows

package control

import (
	"net"
	"os"
	"path/filepath"

	"net-merge/internal/state"
)

// Listen opens a user-only unix socket in the state directory.
func Listen() (net.Listener, error) {
	dir := state.DefaultDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "multiroute.sock")
	os.Remove(path)
	return net.Listen("unix", path)
}

// Dial connects to that socket.
func Dial() (net.Conn, error) {
	return net.Dial("unix", filepath.Join(state.DefaultDir(), "multiroute.sock"))
}

// IsPipeAbsent permits offline configuration only when no endpoint exists.
func IsPipeAbsent(err error) bool { return os.IsNotExist(err) }

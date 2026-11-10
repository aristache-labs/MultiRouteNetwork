//go:build !windows

package control

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"MultiRouteNetwork/internal/state"
)

const sockName = "multiroute.sock"

// privilegedListen is set by the installed service. The legacy tray keeps the user socket.
var privilegedListen string

// SetListenDir makes Listen use the privileged service directory.
func SetListenDir(dir string) { privilegedListen = dir }

// Listen opens the desktop endpoint. The service sets SetListenDir first.
func Listen() (net.Listener, error) {
	if privilegedListen != "" {
		return ListenPrivileged(privilegedListen)
	}
	return listenUser()
}

func listenUser() (net.Listener, error) {
	dir := state.DefaultDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, sockName)
	_ = os.Remove(path)
	return net.Listen("unix", path)
}

// Dial connects the desktop to the privileged service socket.
func Dial() (net.Conn, error) {
	return DialPath(state.ServiceDir())
}

// DialPath connects to a service directory. Tests and the integration helper use it.
func DialPath(dir string) (net.Conn, error) {
	return net.Dial("unix", filepath.Join(dir, sockName))
}

// IsPipeAbsent permits offline configuration only when no endpoint exists.
func IsPipeAbsent(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOENT)
}

// IsEndpointDown is a leftover socket with no process accepting it.
func IsEndpointDown(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}

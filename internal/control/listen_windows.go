//go:build windows

package control

import (
	"context"
	"errors"
	"net"
	"time"

	winio "github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func userPipePath() (string, string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", "", err
	}
	sid := user.User.Sid.String()
	return `\\.\pipe\MultiRoute-` + sid, sid, nil
}

// Listen keeps the endpoint available while accepting clients, and uses overlapped
// I/O so abandoned connections and Close cannot strand the server's accept loop.
func Listen() (net.Listener, error) {
	path, _, err := userPipePath()
	if err != nil {
		return nil, err
	}
	return listenUserPipe(path)
}

func listenUserPipe(path string) (net.Listener, error) {
	_, sid, err := userPipePath()
	if err != nil {
		return nil, err
	}
	return winio.ListenPipe(path, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;" + sid + ")",
		InputBufferSize:    64 << 10,
		OutputBufferSize:   64 << 10,
	})
}

func Dial() (net.Conn, error) {
	path, _, err := userPipePath()
	if err != nil {
		return nil, err
	}
	return dialPipe(path)
}

func dialPipe(path string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return winio.DialPipeContext(ctx, path)
}

// A missing endpoint is different from an occupied, denied or timed-out one.
func IsPipeAbsent(err error) bool {
	return errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND)
}

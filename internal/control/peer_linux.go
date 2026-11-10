//go:build linux

package control

import (
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

func peerCredentials(c net.Conn) (int, string, error) {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return 0, "", errors.New("not a unix connection")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, "", err
	}
	var ucred *unix.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		ucred, credErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return 0, "", err
	}
	if credErr != nil {
		return 0, "", credErr
	}
	exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", ucred.Pid))
	if err != nil {
		return 0, "", err
	}
	return int(ucred.Uid), exe, nil
}

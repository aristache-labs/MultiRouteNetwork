//go:build !windows

package control

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"time"

	"MultiRouteNetwork/internal/state"
	"golang.org/x/sys/unix"
)

// peerOf is replaced by tests. Production reads the kernel identity.
var peerOf = peerCredentials

// ListenPrivileged binds a socket in a directory the desktop user cannot replace.
func ListenPrivileged(dir string) (net.Listener, error) {
	if err := state.PrepareService(dir); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, sockName)
	if err := unlinkSocket(path); err != nil {
		return nil, err
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o666); err != nil {
		ln.Close()
		return nil, err
	}
	return &gatedListener{Listener: ln, dir: dir}, nil
}

func unlinkSocket(path string) error {
	fi, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return state.ErrSymlink
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return errors.New("service socket path is not a socket")
	}
	return os.Remove(path)
}

type gatedListener struct {
	net.Listener
	dir string
}

func (l *gatedListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		ok, authErr := l.allow(c)
		if authErr != nil {
			c.Close()
			continue
		}
		if ok {
			return c, nil
		}
		go rejectPeer(c)
	}
}

func (l *gatedListener) allow(c net.Conn) (bool, error) {
	uid, exe, err := peerOf(c)
	if err != nil {
		return false, err
	}
	existing, readErr := ReadAuth(l.dir)
	missing := errors.Is(readErr, os.ErrNotExist)
	if readErr != nil && !missing {
		return false, readErr
	}
	next, ok := Authorize(existing, missing, uid, exe, expectedGUI())
	if !ok {
		return false, nil
	}
	if missing {
		if err := WriteAuth(l.dir, next); err != nil {
			return false, err
		}
	}
	return true, nil
}

func rejectPeer(c net.Conn) {
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	var req Request
	_ = json.NewDecoder(c).Decode(&req)
	_ = json.NewEncoder(c).Encode(Reply{Error: ErrAuthorization.Error()})
}

func expectedGUI() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Clean(filepath.Join(filepath.Dir(exe), "MultiRoute"))
}

// ReadAuth loads the authorized desktop user. The file is not trusted if it is a symlink.
func ReadAuth(dir string) (Auth, error) {
	path := filepath.Join(dir, authFileName)
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return Auth{}, os.ErrNotExist
		}
		return Auth{}, err
	}
	defer unix.Close(fd)
	fi, err := os.Lstat(path)
	if err != nil {
		return Auth{}, err
	}
	buf := make([]byte, fi.Size())
	n, err := unix.Read(fd, buf)
	if err != nil {
		return Auth{}, err
	}
	return ParseAuth(string(buf[:n]))
}

// WriteAuth stores the authorized user without following a symlink.
func WriteAuth(dir string, a Auth) error {
	if err := state.PrepareService(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, authFileName)
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC|unix.O_NOFOLLOW, 0o644)
	if err != nil {
		return err
	}
	text := FormatAuth(a)
	if _, err := unix.Write(fd, []byte(text)); err != nil {
		unix.Close(fd)
		return err
	}
	return unix.Close(fd)
}

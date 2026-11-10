package control

import (
	"errors"
	"strconv"
	"strings"
)

const authFileName = "authorized-user"

// Auth is the single user allowed to control the privileged service.
type Auth struct {
	UID int
	Exe string
}

// ErrAuthorization is returned when the peer is not the authorized desktop user.
var ErrAuthorization = errors.New("authorization required")

// ParseAuth reads the uid and executable stored for the service.
func ParseAuth(text string) (Auth, error) {
	var a Auth
	var sawUID, sawExe bool
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.ContainsAny(val, "\r\n") {
			return Auth{}, errors.New("invalid authorized user")
		}
		switch key {
		case "uid":
			n, err := strconv.Atoi(val)
			if err != nil || n < 0 || sawUID {
				return Auth{}, errors.New("invalid authorized user")
			}
			a.UID = n
			sawUID = true
		case "exe":
			if val == "" || sawExe || strings.Contains(val, "\x00") {
				return Auth{}, errors.New("invalid authorized user")
			}
			a.Exe = val
			sawExe = true
		default:
			return Auth{}, errors.New("invalid authorized user")
		}
	}
	if !sawUID || !sawExe {
		return Auth{}, errors.New("invalid authorized user")
	}
	return a, nil
}

// FormatAuth is the file contents written by the installer.
func FormatAuth(a Auth) string {
	return "uid=" + strconv.Itoa(a.UID) + "\nexe=" + a.Exe + "\n"
}

// Authorize decides whether a kernel-reported peer may control the service.
// A missing record can be claimed once by the installed GUI, never by root and never from the request body.
func Authorize(existing Auth, missing bool, uid int, exe, expectedGUI string) (Auth, bool) {
	if uid <= 0 || exe == "" || strings.Contains(exe, " (deleted)") {
		return Auth{}, false
	}
	if missing {
		if expectedGUI == "" || exe != expectedGUI {
			return Auth{}, false
		}
		return Auth{UID: uid, Exe: exe}, true
	}
	if uid == existing.UID && exe == existing.Exe {
		return existing, true
	}
	return Auth{}, false
}

package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

const BundleID = "dev.aristache.multiroute"

const (
	linuxServiceDir = "/var/lib/multiroute"
	macServiceDir   = "/Library/Application Support/dev.aristache.multiroute"
)

// ErrSymlink refuses a service directory or socket that is a symlink.
var ErrSymlink = errors.New("service path is a symlink")

// ServiceDir is the root-owned directory for the engine database and socket.
// Windows keeps using DefaultDir so existing installs stay where they are.
func ServiceDir() string {
	switch runtime.GOOS {
	case "linux":
		return linuxServiceDir
	case "darwin":
		return macServiceDir
	default:
		return DefaultDir()
	}
}

// PreferenceDir holds window preferences. It is not the privileged service directory.
func PreferenceDir() string {
	if runtime.GOOS == "windows" {
		return DefaultDir()
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "MultiRoute")
}

// PrepareService creates dir without following a symlink at the leaf or its parent.
func PrepareService(dir string) error {
	if err := rejectServiceSymlink(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.Chmod(dir, 0o755)
}

func rejectServiceSymlink(dir string) error {
	for _, path := range []string{dir, filepath.Dir(dir)} {
		fi, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return ErrSymlink
		}
	}
	return nil
}

//go:build darwin

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"MultiRouteNetwork/internal/control"
	"MultiRouteNetwork/internal/state"
)

/*
#cgo LDFLAGS: -framework ServiceManagement -framework Foundation
#include <stdlib.h>
int mr_register_daemon(const char *plist, char *errbuf, int errlen);
*/
import "C"
import "unsafe"

func launchEngine() error {
	engine := enginePath()
	if _, err := os.Stat(engine); err != nil {
		return errNoEngine
	}
	gui, err := os.Executable()
	if err != nil {
		return err
	}
	if !serviceInstalled() {
		script := "do shell script " + shellQuote(engine+" service-install --uid "+strconv.Itoa(os.Getuid())+" --gui "+gui) + " with administrator privileges"
		if out, err := exec.Command("osascript", "-e", script).CombinedOutput(); err != nil {
			return fmt.Errorf("%w: %s", control.ErrAuthorization, out)
		}
	}
	if err := registerDaemon(); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "kickstart", "-k", "system/"+state.BundleID+".daemon").Run()
	return waitForRouting()
}

func serviceInstalled() bool {
	_, err := os.Stat("/Library/Application Support/dev.aristache.multiroute/authorized-user")
	if err == nil {
		return true
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Clean(filepath.Join(filepath.Dir(exe), "..", "Library", "LaunchDaemons", state.BundleID+".daemon.plist")))
	return err == nil
}

func registerDaemon() error {
	name := state.BundleID + ".daemon.plist"
	buf := make([]byte, 512)
	cstr := C.CString(name)
	defer C.free(unsafe.Pointer(cstr))
	if C.mr_register_daemon(cstr, (*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf))) != 0 {
		text := C.GoString((*C.char)(unsafe.Pointer(&buf[0])))
		if strings.Contains(strings.ToLower(text), "auth") || strings.Contains(strings.ToLower(text), "cancel") {
			return fmt.Errorf("%w: %s", control.ErrAuthorization, text)
		}
		return errors.New(text)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func waitForRouting() error {
	deadline := time.Now().Add(engineStartupTimeout)
	var last error
	for time.Now().Before(deadline) {
		conn, err := control.Dial()
		if err != nil {
			last = err
			time.Sleep(200 * time.Millisecond)
			continue
		}
		_, err = control.Call(conn, control.Request{Op: "start"})
		return err
	}
	if last == nil {
		last = errors.New("service did not accept a connection")
	}
	return fmt.Errorf("The routing engine did not become ready within 30 seconds. Try Start again: %w", last)
}

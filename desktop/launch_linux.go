//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"MultiRouteNetwork/internal/control"
)

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
		cmd := exec.Command("pkexec", engine, "service-install", "--uid", strconv.Itoa(os.Getuid()), "--gui", gui)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%w: %s", control.ErrAuthorization, err.Error())
		}
	}
	if out, err := exec.Command("systemctl", "start", "multiroute.service").CombinedOutput(); err != nil {
		return fmt.Errorf("start service: %s", out)
	}
	return waitForRouting()
}

func serviceInstalled() bool {
	if _, err := os.Stat("/etc/systemd/system/multiroute.service"); err == nil {
		return true
	}
	if _, err := os.Stat("/usr/lib/systemd/system/multiroute.service"); err == nil {
		return true
	}
	_, err := os.Stat("/var/lib/multiroute/authorized-user")
	return err == nil
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

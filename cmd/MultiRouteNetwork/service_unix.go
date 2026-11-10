//go:build !windows

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"MultiRouteNetwork/internal/control"
	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/logx"
	"MultiRouteNetwork/internal/platform"
	"MultiRouteNetwork/internal/state"
	"MultiRouteNetwork/internal/tun"
)

func cmdService(args []string) {
	fs := flag.NewFlagSet("service", flag.ExitOnError)
	dir := fs.String("state-dir", state.ServiceDir(), "privileged service directory")
	fs.Parse(args)
	if os.Geteuid() != 0 {
		die(fmt.Errorf("authorization required"))
	}
	if err := state.PrepareService(*dir); err != nil {
		die(err)
	}
	if !platform.SingleInstance() {
		die(fmt.Errorf("MultiRouteNetwork service is already running"))
	}
	control.EnableStayResident()
	control.SetListenDir(*dir)
	c := common{weights: weightFlags{}, rescan: 3 * time.Second, auto: true, ipv6: "auto", mtu: tun.DefaultMTU, stateDir: *dir}
	sched, bal, err := c.boot(func() string { return "desktop-service" })
	var sw control.Switch
	if err != nil {
		logx.Printf("warning: %v\n", err)
		sched = engine.NewScheduler(nil)
		bal = engine.NewBalancer(sched)
	}
	adapter := tun.New(bal)
	sw = adapter
	svc := hostControl(*dir, sched, bal, sw)
	if svc == nil {
		die(fmt.Errorf("service control did not start"))
	}
	logx.Printf("MultiRoute service is ready. Routing stays off until Start.\n")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	adapter.Close()
	logx.Printf("MultiRoute service stopped.\n")
}

func cmdServiceInstall(args []string) {
	fs := flag.NewFlagSet("service-install", flag.ExitOnError)
	uid := fs.Int("uid", -1, "desktop user id")
	gui := fs.String("gui", "", "desktop executable")
	dir := fs.String("state-dir", state.ServiceDir(), "privileged service directory")
	allowRoot := fs.Bool("allow-root", false, "test only: authorize uid 0")
	noUnit := fs.Bool("no-unit", false, "do not install or start the system unit")
	fs.Parse(args)
	if os.Geteuid() != 0 {
		die(fmt.Errorf("authorization required"))
	}
	if *uid < 0 || *gui == "" {
		die(fmt.Errorf("service-install needs --uid and --gui"))
	}
	if *uid == 0 && !*allowRoot {
		die(fmt.Errorf("authorization required"))
	}
	if err := state.PrepareService(*dir); err != nil {
		die(err)
	}
	if err := control.WriteAuth(*dir, control.Auth{UID: *uid, Exe: *gui}); err != nil {
		die(err)
	}
	if *noUnit || runtime.GOOS != "linux" {
		return
	}
	if err := installLinuxUnit(); err != nil {
		die(err)
	}
}

func cmdServiceCall(args []string) {
	fs := flag.NewFlagSet("service-call", flag.ExitOnError)
	dir := fs.String("state-dir", state.ServiceDir(), "privileged service directory")
	op := fs.String("op", "status", "status, start, or stop")
	fs.Parse(args)
	switch *op {
	case "status", "start", "stop":
	default:
		die(fmt.Errorf("unsupported service operation"))
	}
	conn, err := control.DialPath(*dir)
	if err != nil {
		die(err)
	}
	rep, err := control.Call(conn, control.Request{Op: *op})
	if err != nil {
		die(err)
	}
	if *op == "status" && rep.Status != nil {
		fmt.Printf("running=%t\n", rep.Status.Running)
	}
}

func installLinuxUnit() error {
	const packaged = "/usr/lib/systemd/system/multiroute.service"
	const dest = "/etc/systemd/system/multiroute.service"
	if _, err := os.Stat(packaged); err != nil {
		if _, err := os.Stat(dest); err != nil {
			if err := os.WriteFile(dest, []byte(linuxUnit), 0o644); err != nil {
				return err
			}
		}
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %s", out)
	}
	if out, err := exec.Command("systemctl", "enable", "--now", "multiroute.service").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable: %s", out)
	}
	return nil
}

const linuxUnit = `[Unit]
Description=MultiRoute routing service
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/MultiRouteNetwork service
Restart=on-failure

[Install]
WantedBy=multi-user.target
`

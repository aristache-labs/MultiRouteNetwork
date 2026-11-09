package tun

import (
	"fmt"
	"golang.org/x/sys/windows"
	"unsafe"
)

var routeAPI = windows.NewLazySystemDLL("iphlpapi.dll")
var createRoute = routeAPI.NewProc("CreateIpForwardEntry2")
var deleteRoute = routeAPI.NewProc("DeleteIpForwardEntry2")

func init() { pauseCapture = pauseWindowsCapture }

// Save precisely the capture routes on this TUN. Native IP Helper calls avoid
// spawning route.exe/netsh for every pause/resume; addresses and MTU stay intact.
func pauseWindowsCapture(idx int) (func() error, error) {
	var table *windows.MibIpForwardTable2
	if err := windows.GetIpForwardTable2(windows.AF_UNSPEC, &table); err != nil {
		return nil, err
	}
	var rows []windows.MibIpForwardRow2
	for _, row := range table.Rows() {
		if row.InterfaceIndex == uint32(idx) && row.DestinationPrefix.PrefixLength == 1 {
			rows = append(rows, row)
		}
	}
	windows.FreeMibTable(unsafe.Pointer(table))
	return pauseRouteRows(idx, rows, func(add bool, row *windows.MibIpForwardRow2) error {
		if add {
			return changeCaptureRoute(createRoute, row)
		}
		return changeCaptureRoute(deleteRoute, row)
	}, startStrongHost, stopStrongHost)
}

func pauseRouteRows(idx int, rows []windows.MibIpForwardRow2, change func(bool, *windows.MibIpForwardRow2) error, start func(int), stop func()) (func() error, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("capture routes missing for interface %d", idx)
	}
	for i := range rows {
		if err := change(false, &rows[i]); err != nil {
			return nil, err
		}
	}
	stop()
	return func() error {
		// Recheck egress safety before capturing any new packets.
		start(idx)
		for i := range rows {
			if err := change(true, &rows[i]); err != nil {
				for j := 0; j < i; j++ {
					_ = change(false, &rows[j])
				}
				stop()
				return err
			}
		}
		return nil
	}, nil
}

func changeCaptureRoute(proc *windows.LazyProc, row *windows.MibIpForwardRow2) error {
	code, _, _ := proc.Call(uintptr(unsafe.Pointer(row)))
	if code != 0 {
		return fmt.Errorf("%s: %w", proc.Name, windows.Errno(code))
	}
	return nil
}

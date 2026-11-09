//go:build windows

package procid

import (
	"golang.org/x/sys/windows"
	"sync"
	"time"
	"unsafe"
)

// Running takes a fresh process snapshot, independent of network activity.
var runningMu sync.Mutex
var runningAt time.Time
var runningList []Proc
var runningErr error

func Running() ([]Proc, error) {
	runningMu.Lock()
	defer runningMu.Unlock()
	if time.Since(runningAt) < time.Second {
		return runningList, runningErr
	}
	runningList, runningErr = scanRunning()
	runningAt = time.Now()
	return runningList, runningErr
}
func scanRunning() ([]Proc, error) {
	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(h)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	err = windows.Process32First(h, &entry)
	out := []Proc{}
	for err == nil {
		p := Proc{PID: entry.ProcessID, Name: windows.UTF16ToString(entry.ExeFile[:]), Known: true}
		ph, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, p.PID)
		if e == nil {
			buf := make([]uint16, 32768)
			n := uint32(len(buf))
			if windows.QueryFullProcessImageName(ph, 0, &buf[0], &n) == nil {
				p.Path = windows.UTF16ToString(buf[:n])
			}
			windows.CloseHandle(ph)
		}
		out = append(out, p)
		err = windows.Process32Next(h, &entry)
	}
	if err != windows.ERROR_NO_MORE_FILES {
		return nil, err
	}
	return out, nil
}

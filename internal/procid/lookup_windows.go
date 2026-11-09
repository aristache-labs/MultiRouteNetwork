//go:build windows

package procid

import (
	"net/netip"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Proc is the owner of a socket. Known is false when the table has no row or the process has exited.
type Proc struct {
	PID   uint32
	Name  string
	Path  string
	Known bool
}

var (
	iphlpapi             = syscall.NewLazyDLL("iphlpapi.dll")
	procGetExtendedTcp   = iphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUdp   = iphlpapi.NewProc("GetExtendedUdpTable")
	procQueryFullProcess = windows.NewLazySystemDLL("kernel32.dll").NewProc("QueryFullProcessImageNameW")
)

const (
	afInet  = 2
	afInet6 = 23
	tcpAll  = 5
	udpOwn  = 1
)

type cacheEntry struct {
	name, path string
	ok         bool
}

var procCache sync.Map

// Lookup maps a local TCP or UDP endpoint to a process. Unknown stays unknown.
func Lookup(network string, addr netip.Addr, port uint16) Proc {
	udp := len(network) >= 3 && network[:3] == "udp"
	rows := table(addr.Is6(), udp)
	pid, ok := Match(rows, addr, port)
	if !ok {
		return Proc{}
	}
	name, path, ok := image(pid)
	if !ok {
		return Proc{PID: pid}
	}
	return Proc{PID: pid, Name: name, Path: path, Known: true}
}

func table(v6, udp bool) []Row {
	fam := uint32(afInet)
	if v6 {
		fam = afInet6
	}
	call := procGetExtendedTcp
	class := uintptr(tcpAll)
	if udp {
		call = procGetExtendedUdp
		class = udpOwn
	}
	var size uint32
	call.Call(0, uintptr(unsafe.Pointer(&size)), 0, uintptr(fam), class, 0)
	if size == 0 || size > 32<<20 {
		return nil
	}
	buf := make([]byte, size)
	r, _, _ := call.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, uintptr(fam), class, 0)
	if r != 0 {
		return nil
	}
	if udp {
		return parseUDP(buf, v6)
	}
	return parseTCP(buf, v6)
}

func parseTCP(buf []byte, v6 bool) []Row {
	if len(buf) < 4 {
		return nil
	}
	n := int(*(*uint32)(unsafe.Pointer(&buf[0])))
	stride := 24
	if v6 {
		stride = 56
	}
	rows := make([]Row, 0, n)
	off := 4
	addrAt, portAt := 4, 8
	if v6 {
		addrAt, portAt = 0, 20
	}
	for i := 0; i < n && off+stride <= len(buf); i++ {
		row := buf[off : off+stride]
		local, port := readEndpoint(row, v6, addrAt, portAt)
		pid := *(*uint32)(unsafe.Pointer(&row[stride-4]))
		rows = append(rows, Row{Local: local, Port: port, PID: pid, HasAddr: true})
		off += stride
	}
	return rows
}

func parseUDP(buf []byte, v6 bool) []Row {
	if len(buf) < 4 {
		return nil
	}
	n := int(*(*uint32)(unsafe.Pointer(&buf[0])))
	stride := 12
	if v6 {
		stride = 28
	}
	rows := make([]Row, 0, n)
	off := 4
	portAt := 4
	if v6 {
		portAt = 20
	}
	for i := 0; i < n && off+stride <= len(buf); i++ {
		row := buf[off : off+stride]
		local, port := readEndpoint(row, v6, 0, portAt)
		pid := *(*uint32)(unsafe.Pointer(&row[stride-4]))
		any := !local.IsValid() || local.IsUnspecified()
		rows = append(rows, Row{Local: local, Port: port, PID: pid, HasAddr: !any})
		off += stride
	}
	return rows
}

func readEndpoint(row []byte, v6 bool, addrAt, portAt int) (netip.Addr, uint16) {
	port := uint16(row[portAt])<<8 | uint16(row[portAt+1])
	if v6 {
		var b [16]byte
		copy(b[:], row[addrAt:addrAt+16])
		return netip.AddrFrom16(b), port
	}
	ip := netip.AddrFrom4(*(*[4]byte)(unsafe.Pointer(&row[addrAt])))
	return ip, port
}

func image(pid uint32) (string, string, bool) {
	if v, ok := procCache.Load(pid); ok {
		e := v.(cacheEntry)
		return e.name, e.path, e.ok
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		procCache.Store(pid, cacheEntry{})
		return "", "", false
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, syscall.MAX_PATH)
	n := uint32(len(buf))
	r, _, _ := procQueryFullProcess.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		procCache.Store(pid, cacheEntry{})
		return "", "", false
	}
	path := syscall.UTF16ToString(buf[:n])
	e := cacheEntry{name: filepath.Base(path), path: path, ok: path != ""}
	procCache.Store(pid, e)
	return e.name, e.path, e.ok
}

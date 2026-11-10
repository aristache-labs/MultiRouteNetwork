package procid

import (
	"net/netip"
	"path/filepath"
	"strconv"
	"strings"
)

// procFiles is the slice of procfs the Linux owner lookup needs.
type procFiles interface {
	Read(path string) ([]byte, error)
	Link(path string) (string, error)
	List(path string) ([]string, error)
}

type ownerCache map[string]procCacheEntry

type procCacheEntry struct {
	name, path string
	ok         bool
}

// LookupProc resolves a local endpoint from a procfs snapshot.
// Two different processes owning the same socket, or a PID whose start time changed, stay unknown.
func LookupProc(fs procFiles, cache ownerCache, network string, addr netip.Addr, port uint16) Proc {
	if cache == nil {
		cache = ownerCache{}
	}
	udp := strings.HasPrefix(network, "udp")
	name := "/proc/net/tcp"
	if addr.Is6() {
		name = "/proc/net/tcp6"
	}
	if udp {
		name = "/proc/net/udp"
		if addr.Is6() {
			name = "/proc/net/udp6"
		}
	}
	text, err := fs.Read(name)
	if err != nil {
		return Proc{}
	}
	inode, ok := inodeFor(string(text), addr, port)
	if !ok {
		return Proc{}
	}
	pids := socketOwners(fs, inode)
	if len(pids) != 1 {
		return Proc{}
	}
	pid := pids[0]
	start, ok := processStart(fs, pid)
	if !ok {
		return Proc{}
	}
	key := strconv.FormatUint(uint64(pid), 10) + ":" + strconv.FormatUint(start, 10)
	if e, hit := cache[key]; hit {
		return Proc{PID: pid, Start: start, Name: e.name, Path: e.path, Known: e.ok}
	}
	link, err := fs.Link("/proc/" + strconv.FormatUint(uint64(pid), 10) + "/exe")
	if err != nil || link == "" || strings.Contains(link, " (deleted)") {
		cache[key] = procCacheEntry{}
		return Proc{PID: pid, Start: start}
	}
	e := procCacheEntry{name: filepath.Base(link), path: link, ok: true}
	cache[key] = e
	return Proc{PID: pid, Start: start, Name: e.name, Path: e.path, Known: true}
}

// ListRunning returns processes whose executable path can be read.
func ListRunning(fs procFiles) ([]Proc, error) {
	names, err := fs.List("/proc")
	if err != nil {
		return nil, err
	}
	out := []Proc{}
	for _, name := range names {
		pid64, err := strconv.ParseUint(name, 10, 32)
		if err != nil {
			continue
		}
		pid := uint32(pid64)
		start, ok := processStart(fs, pid)
		if !ok {
			continue
		}
		link, err := fs.Link("/proc/" + name + "/exe")
		if err != nil || link == "" || strings.Contains(link, " (deleted)") {
			continue
		}
		out = append(out, Proc{PID: pid, Start: start, Name: filepath.Base(link), Path: link, Known: true})
	}
	return out, nil
}

func inodeFor(table string, addr netip.Addr, port uint16) (string, bool) {
	addr = addr.Unmap()
	var inode string
	found := false
	for _, line := range strings.Split(table, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 || !strings.Contains(fields[1], ":") {
			continue
		}
		local, p, ok := parseProcEndpoint(fields[1])
		if !ok || p != port {
			continue
		}
		if local.IsValid() && local.Unmap() == addr {
			return fields[9], true
		}
		if (!local.IsValid() || local.IsUnspecified()) && !found {
			inode, found = fields[9], true
		}
	}
	return inode, found
}

func parseProcEndpoint(field string) (netip.Addr, uint16, bool) {
	ipHex, portHex, ok := strings.Cut(field, ":")
	if !ok {
		return netip.Addr{}, 0, false
	}
	port64, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return netip.Addr{}, 0, false
	}
	if len(ipHex) == 8 {
		word, err := strconv.ParseUint(ipHex, 16, 32)
		if err != nil {
			return netip.Addr{}, 0, false
		}
		ip := uint32(word)
		var b [4]byte
		b[0], b[1], b[2], b[3] = byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24)
		return netip.AddrFrom4(b), uint16(port64), true
	}
	if len(ipHex) == 32 {
		var b [16]byte
		for i := 0; i < 4; i++ {
			word, err := strconv.ParseUint(ipHex[i*8:(i+1)*8], 16, 32)
			if err != nil {
				return netip.Addr{}, 0, false
			}
			ip := uint32(word)
			b[i*4], b[i*4+1], b[i*4+2], b[i*4+3] = byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24)
		}
		return netip.AddrFrom16(b), uint16(port64), true
	}
	return netip.Addr{}, 0, false
}

func processStart(fs procFiles, pid uint32) (uint64, bool) {
	text, err := fs.Read("/proc/" + strconv.FormatUint(uint64(pid), 10) + "/stat")
	if err != nil {
		return 0, false
	}
	return statStart(string(text))
}

func statStart(stat string) (uint64, bool) {
	end := strings.LastIndex(stat, ")")
	if end < 0 || end+2 >= len(stat) {
		return 0, false
	}
	fields := strings.Fields(stat[end+2:])
	// Field 22 of the whole stat line is starttime. Fields before comm are "pid (comm)",
	// so starttime is index 19 of the remainder (field 3 is index 0).
	if len(fields) < 20 {
		return 0, false
	}
	n, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return n, true
}

func socketOwners(fs procFiles, inode string) []uint32 {
	names, err := fs.List("/proc")
	if err != nil {
		return nil
	}
	want := "socket:[" + inode + "]"
	var pids []uint32
	seen := map[uint32]bool{}
	for _, name := range names {
		pid64, err := strconv.ParseUint(name, 10, 32)
		if err != nil {
			continue
		}
		fds, err := fs.List("/proc/" + name + "/fd")
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := fs.Link("/proc/" + name + "/fd/" + fd)
			if err != nil || link != want {
				continue
			}
			pid := uint32(pid64)
			if !seen[pid] {
				seen[pid] = true
				pids = append(pids, pid)
			}
		}
	}
	return pids
}

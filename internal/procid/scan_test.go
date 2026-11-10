package procid

import (
	"net/netip"
	"strconv"
	"testing"
)

type memProc struct {
	files map[string]string
	links map[string]string
	dirs  map[string][]string
}

func (m memProc) Read(path string) ([]byte, error) {
	text, ok := m.files[path]
	if !ok {
		return nil, osErr(path)
	}
	return []byte(text), nil
}
func (m memProc) Link(path string) (string, error) {
	link, ok := m.links[path]
	if !ok {
		return "", osErr(path)
	}
	return link, nil
}
func (m memProc) List(path string) ([]string, error) {
	names, ok := m.dirs[path]
	if !ok {
		return nil, osErr(path)
	}
	return names, nil
}

type missing string

func (m missing) Error() string { return "missing " + string(m) }

func osErr(path string) error { return missing(path) }

func statLine(pid uint32, start uint64) string {
	return strconv.FormatUint(uint64(pid), 10) + " (curl) S 1 1 1 0 -1 0 0 0 0 0 0 0 0 0 20 0 1 0 " + strconv.FormatUint(start, 10)
}

func TestLookupProcNamesTheSocketOwner(t *testing.T) {
	fs := memProc{
		files: map[string]string{
			"/proc/net/tcp": "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 0 100 0 0 10 0\n",
			"/proc/10/stat": statLine(10, 100),
		},
		links: map[string]string{"/proc/10/fd/3": "socket:[12345]", "/proc/10/exe": "/usr/bin/curl"},
		dirs:  map[string][]string{"/proc": {"10"}, "/proc/10/fd": {"3"}},
	}
	got := LookupProc(fs, ownerCache{}, "tcp", netip.MustParseAddr("127.0.0.1"), 8080)
	if !got.Known || got.PID != 10 || got.Start != 100 || got.Path != "/usr/bin/curl" || got.Name != "curl" {
		t.Fatalf("%+v", got)
	}
}

func TestLookupProcDoesNotReuseAPID(t *testing.T) {
	fs := memProc{
		files: map[string]string{
			"/proc/net/tcp": "  0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 12345 1 0 100 0\n",
			"/proc/10/stat": statLine(10, 100),
		},
		links: map[string]string{"/proc/10/fd/3": "socket:[12345]", "/proc/10/exe": "/usr/bin/curl"},
		dirs:  map[string][]string{"/proc": {"10"}, "/proc/10/fd": {"3"}},
	}
	cache := ownerCache{}
	first := LookupProc(fs, cache, "tcp", netip.MustParseAddr("127.0.0.1"), 8080)
	fs.files["/proc/10/stat"] = statLine(10, 200)
	fs.links["/proc/10/exe"] = "/usr/bin/wget"
	second := LookupProc(fs, cache, "tcp", netip.MustParseAddr("127.0.0.1"), 8080)
	if !SameProcess(first, first) || SameProcess(first, second) {
		t.Fatal("start time did not separate the processes")
	}
	if second.Path != "/usr/bin/wget" || second.Start != 200 || first.Path != "/usr/bin/curl" {
		t.Fatalf("reused identity: %+v then %+v", first, second)
	}
}

func TestLookupProcStaysUnknownWhenAmbiguousOrDeleted(t *testing.T) {
	fs := memProc{
		files: map[string]string{
			"/proc/net/udp": "  0: 00000000:0035 00000000:0000 07 00000000:00000000 00:00000000 00000000 0 0 99 1 0 100 0\n",
			"/proc/4/stat":  statLine(4, 5),
			"/proc/5/stat":  statLine(5, 6),
		},
		links: map[string]string{
			"/proc/4/fd/1": "socket:[99]",
			"/proc/5/fd/1": "socket:[99]",
			"/proc/4/exe":  "/usr/bin/a",
			"/proc/5/exe":  "/usr/bin/b",
		},
		dirs: map[string][]string{"/proc": {"4", "5"}, "/proc/4/fd": {"1"}, "/proc/5/fd": {"1"}},
	}
	if got := LookupProc(fs, nil, "udp", netip.MustParseAddr("127.0.0.1"), 53); got.Known {
		t.Fatalf("ambiguous owner: %+v", got)
	}
	fs.dirs["/proc"] = []string{"4"}
	fs.links["/proc/4/exe"] = "/usr/bin/a (deleted)"
	if got := LookupProc(fs, ownerCache{}, "udp", netip.MustParseAddr("127.0.0.1"), 53); got.Known {
		t.Fatalf("deleted executable: %+v", got)
	}
}

func TestListRunningSkipsUnreadableExecutables(t *testing.T) {
	fs := memProc{
		files: map[string]string{"/proc/7/stat": statLine(7, 9), "/proc/8/stat": statLine(8, 9)},
		links: map[string]string{"/proc/7/exe": "/usr/bin/curl"},
		dirs:  map[string][]string{"/proc": {"7", "8", "self"}},
	}
	got, err := ListRunning(fs)
	if err != nil || len(got) != 1 || got[0].Path != "/usr/bin/curl" {
		t.Fatalf("%v %+v", err, got)
	}
}

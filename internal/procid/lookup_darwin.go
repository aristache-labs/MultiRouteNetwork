//go:build darwin

package procid

/*
#include <libproc.h>
#include <stdlib.h>
#include <string.h>
#include <arpa/inet.h>
#include <netinet/in.h>

typedef struct {
	int pid;
	unsigned long long start;
	int family;
	int port;
	int ok;
	unsigned char addr[16];
	char path[PROC_PIDPATHINFO_MAXSIZE];
} mr_owner;

static int address_matches(struct in_sockinfo *in, int want4, const unsigned char *addr, int addrlen, int port) {
	if (ntohs((unsigned short)in->insi_lport) != (unsigned short)port)
		return 0;
	if (want4 && (in->insi_vflag & INI_IPV4) && addrlen == 4)
		return memcmp(&in->insi_laddr.ina_46.i46a_addr4, addr, 4) == 0;
	if (!want4 && (in->insi_vflag & INI_IPV6) && addrlen == 16)
		return memcmp(&in->insi_laddr.ina_6, addr, 16) == 0;
	return 0;
}

static void remember(mr_owner *hit, int pid) {
	struct proc_bsdinfo info;
	hit->pid = pid;
	hit->ok = 1;
	if (proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &info, sizeof(info)) > 0)
		hit->start = (unsigned long long)info.pbi_start_tvsec;
	proc_pidpath(pid, hit->path, sizeof(hit->path));
}

int mr_lookup(int want4, const unsigned char *addr, int addrlen, int port, int udp, mr_owner *out) {
	int bytes = proc_listpids(PROC_ALL_PIDS, 0, NULL, 0);
	if (bytes <= 0)
		return 0;
	int count = bytes / (int)sizeof(int);
	int *pids = malloc((size_t)bytes);
	if (!pids)
		return 0;
	bytes = proc_listpids(PROC_ALL_PIDS, 0, pids, bytes);
	count = bytes / (int)sizeof(int);
	int matches = 0;
	mr_owner found;
	memset(&found, 0, sizeof(found));
	for (int i = 0; i < count; i++) {
		int pid = pids[i];
		if (pid <= 0)
			continue;
		int fdbytes = proc_pidinfo(pid, PROC_PIDLISTFDS, 0, NULL, 0);
		if (fdbytes <= 0)
			continue;
		struct proc_fdinfo *fds = malloc((size_t)fdbytes);
		if (!fds)
			continue;
		fdbytes = proc_pidinfo(pid, PROC_PIDLISTFDS, 0, fds, fdbytes);
		int nfds = fdbytes / (int)sizeof(struct proc_fdinfo);
		for (int f = 0; f < nfds; f++) {
			if (fds[f].proc_fdtype != PROX_FDTYPE_SOCKET)
				continue;
			struct socket_fdinfo si;
			int n = proc_pidfdinfo(pid, fds[f].proc_fd, PROC_PIDFDSOCKETINFO, &si, sizeof(si));
			if (n <= 0)
				continue;
			struct in_sockinfo *in = NULL;
			if (!udp && si.psi.soi_kind == SOCKINFO_TCP)
				in = &si.psi.soi_proto.pri_tcp.tcpsi_ini;
			else if (udp && si.psi.soi_kind == SOCKINFO_IN)
				in = &si.psi.soi_proto.pri_in;
			if (!in || !address_matches(in, want4, addr, addrlen, port))
				continue;
			if (matches == 0)
				remember(&found, pid);
			matches++;
		}
		free(fds);
	}
	free(pids);
	if (matches != 1 || found.start == 0 || found.path[0] == '\0')
		return 0;
	*out = found;
	return 1;
}

typedef struct {
	int pid;
	unsigned long long start;
	char path[PROC_PIDPATHINFO_MAXSIZE];
} mr_proc;

int mr_list(mr_proc *out, int cap) {
	int bytes = proc_listpids(PROC_ALL_PIDS, 0, NULL, 0);
	if (bytes <= 0)
		return 0;
	int *pids = malloc((size_t)bytes);
	if (!pids)
		return 0;
	bytes = proc_listpids(PROC_ALL_PIDS, 0, pids, bytes);
	int count = bytes / (int)sizeof(int);
	int n = 0;
	for (int i = 0; i < count && n < cap; i++) {
		int pid = pids[i];
		if (pid <= 0)
			continue;
		struct proc_bsdinfo info;
		if (proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &info, sizeof(info)) <= 0)
			continue;
		if (info.pbi_start_tvsec == 0)
			continue;
		if (proc_pidpath(pid, out[n].path, sizeof(out[n].path)) <= 0)
			continue;
		out[n].pid = pid;
		out[n].start = (unsigned long long)info.pbi_start_tvsec;
		n++;
	}
	free(pids);
	return n;
}
*/
import "C"
import (
	"net/netip"
	"path/filepath"
	"unsafe"
)

// Lookup maps a local TCP or UDP endpoint through libproc.
// Ambiguous matches and processes without a start time stay unknown.
func Lookup(network string, addr netip.Addr, port uint16) Proc {
	addr = addr.Unmap()
	want4 := !addr.Is6()
	var raw []byte
	if want4 {
		b := addr.As4()
		raw = b[:]
	} else {
		b := addr.As16()
		raw = b[:]
	}
	udp := 0
	if len(network) >= 3 && network[:3] == "udp" {
		udp = 1
	}
	var hit C.mr_owner
	ok := C.mr_lookup(C.int(boolInt(want4)), (*C.uchar)(unsafe.Pointer(&raw[0])), C.int(len(raw)), C.int(port), C.int(udp), &hit)
	if ok != 1 {
		return Proc{}
	}
	path := C.GoString(&hit.path[0])
	return Proc{PID: uint32(hit.pid), Start: uint64(hit.start), Name: filepath.Base(path), Path: path, Known: path != ""}
}

// Running lists processes libproc can name.
func Running() ([]Proc, error) {
	const capN = 8192
	buf := make([]C.mr_proc, capN)
	n := int(C.mr_list(&buf[0], C.int(capN)))
	out := make([]Proc, 0, n)
	for i := 0; i < n; i++ {
		path := C.GoString(&buf[i].path[0])
		if path == "" || buf[i].start == 0 {
			continue
		}
		out = append(out, Proc{PID: uint32(buf[i].pid), Start: uint64(buf[i].start), Name: filepath.Base(path), Path: path, Known: true})
	}
	return out, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

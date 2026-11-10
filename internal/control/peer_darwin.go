//go:build darwin

package control

/*
#include <libproc.h>
#include <sys/types.h>
#include <sys/un.h>
#include <unistd.h>
*/
import "C"
import (
	"errors"
	"net"
	"unsafe"

	"golang.org/x/sys/unix"
)

func peerCredentials(c net.Conn) (int, string, error) {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return 0, "", errors.New("not a unix connection")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, "", err
	}
	var pid, uid int
	var ctrlErr error
	if err := raw.Control(func(fd uintptr) {
		pid, ctrlErr = unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
		if ctrlErr != nil {
			return
		}
		var euid C.uid_t
		var egid C.gid_t
		if C.getpeereid(C.int(fd), &euid, &egid) != 0 {
			ctrlErr = errors.New("getpeereid")
			return
		}
		uid = int(euid)
	}); err != nil {
		return 0, "", err
	}
	if ctrlErr != nil {
		return 0, "", ctrlErr
	}
	buf := make([]byte, C.PROC_PIDPATHINFO_MAXSIZE)
	n := C.proc_pidpath(C.int(pid), unsafe.Pointer(&buf[0]), C.uint32_t(len(buf)))
	if n <= 0 {
		return 0, "", errors.New("proc_pidpath")
	}
	return uid, string(buf[:n]), nil
}

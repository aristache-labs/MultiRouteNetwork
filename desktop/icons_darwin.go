//go:build darwin

package main

/*
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include <stdlib.h>
int mr_icon_png(const char *path, unsigned char **out, int *outlen);
*/
import "C"
import (
	"errors"
	"unsafe"
)

func executableIcon(path string) ([]byte, error) {
	cstr := C.CString(path)
	defer C.free(unsafe.Pointer(cstr))
	var out *C.uchar
	var n C.int
	if C.mr_icon_png(cstr, &out, &n) != 0 || n <= 0 {
		return nil, errors.New("no application icon")
	}
	defer C.free(unsafe.Pointer(out))
	return C.GoBytes(unsafe.Pointer(out), n), nil
}

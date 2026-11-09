//go:build windows

package control

import (
	"os"

	"golang.org/x/sys/windows"
)

func osFile(h windows.Handle) *os.File {
	return os.NewFile(uintptr(h), "pipe")
}

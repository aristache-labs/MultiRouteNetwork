//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func serviceInstalled() bool { return false }

func launchEngine() error {
	path := enginePath()
	if _, err := os.Stat(path); err != nil {
		return errNoEngine
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(path)
	params, _ := windows.UTF16PtrFromString("run -hidden")
	dir, _ := windows.UTF16PtrFromString("")
	return windows.ShellExecute(0, verb, file, params, dir, windows.SW_HIDE)
}

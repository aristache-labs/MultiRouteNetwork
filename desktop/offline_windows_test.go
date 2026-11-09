//go:build windows

package main

import "golang.org/x/sys/windows"

func absentPipeError() error { return windows.ERROR_FILE_NOT_FOUND }

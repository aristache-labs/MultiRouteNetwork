//go:build !windows

package main

import "os"

func absentPipeError() error { return os.ErrNotExist }

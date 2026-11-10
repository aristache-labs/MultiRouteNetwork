//go:build windows

package main

import (
	"fmt"
	"os"
)

func cmdService([]string) {
	fmt.Fprintln(os.Stderr, "the desktop service is used on Linux and macOS")
	os.Exit(2)
}

func cmdServiceInstall([]string) { cmdService(nil) }

func cmdServiceCall([]string) { cmdService(nil) }

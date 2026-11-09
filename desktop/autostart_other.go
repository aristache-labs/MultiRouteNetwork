//go:build !windows

package main

import "errors"

func setDesktopAutostart(bool) error {
	return errors.New("Windows startup is only available on Windows")
}

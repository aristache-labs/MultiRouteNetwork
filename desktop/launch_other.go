//go:build !windows

package main

import "errors"

func launchEngine() error {
	return errors.New("the desktop window starts the engine on Windows")
}

//go:build !windows

package main

import "errors"

func executableIcon(path string) ([]byte, error) { return nil, errors.New("Windows icons unavailable") }

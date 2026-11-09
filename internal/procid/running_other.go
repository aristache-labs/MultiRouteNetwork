//go:build !windows

package procid

import "errors"

func Running() ([]Proc, error) { return nil, errors.New("process inventory unavailable") }

//go:build !windows

package nics

// Negotiated rates are currently exposed only by the Windows desktop backend.
func LinkSpeeds(idx int) (receive, transmit uint64) { return 0, 0 }

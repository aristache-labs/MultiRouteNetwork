package tun

import (
	"fmt"
	"io"

	"github.com/xjasonlyu/tun2socks/v2/core/device"
)

func optimizePacketDevice(raw device.Device) (device.Device, error) {
	rw, ok := raw.(io.ReadWriter)
	if !ok {
		return nil, fmt.Errorf("Windows TUN has no packet IO interface")
	}
	return newPacketDevice(raw, rw, 0)
}

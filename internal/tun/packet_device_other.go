//go:build !windows

package tun

import "github.com/xjasonlyu/tun2socks/v2/core/device"

func optimizePacketDevice(raw device.Device) (device.Device, error) { return raw, nil }

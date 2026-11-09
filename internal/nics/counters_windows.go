package nics

import "golang.org/x/sys/windows"

// Octets reads Windows' own byte counters for an adapter. They cover every app's traffic, not
// just what MultiRouteNetwork relays: apps like qBittorrent bind to each adapter directly and bypass the
// TUN, but still show up here.
func Octets(idx int) (in, out uint64, ok bool) {
	if idx <= 0 {
		return 0, 0, false
	}
	row := windows.MibIfRow2{InterfaceIndex: uint32(idx)}
	if windows.GetIfEntry2Ex(windows.MibIfEntryNormal, &row) != nil {
		return 0, 0, false
	}
	return row.InOctets, row.OutOctets, true
}

// LinkSpeeds returns negotiated physical receive/transmit rates in bits/second.
// Virtual adapters and disconnected links must not define the UI's scale.
func LinkSpeeds(idx int) (receive, transmit uint64) {
	if idx <= 0 {
		return 0, 0
	}
	row := windows.MibIfRow2{InterfaceIndex: uint32(idx)}
	if windows.GetIfEntry2Ex(windows.MibIfEntryNormal, &row) != nil || row.OperStatus != 1 || row.MediaConnectState == 2 || !physicalIfType(row.Type) {
		return 0, 0
	}
	if row.ReceiveLinkSpeed != ^uint64(0) {
		receive = row.ReceiveLinkSpeed
	}
	if row.TransmitLinkSpeed != ^uint64(0) {
		transmit = row.TransmitLinkSpeed
	}
	return receive, transmit
}

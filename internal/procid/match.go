package procid

import "net/netip"

// Row is one TCP or UDP owner-table entry. Ports are host byte order.
type Row struct {
	Local   netip.Addr
	Port    uint16
	PID     uint32
	HasAddr bool
}

// Match returns the PID whose local endpoint equals addr:port.
// A row with HasAddr false matches on the port alone, which is how UDP tables look
// when the socket is bound to every address.
func Match(rows []Row, addr netip.Addr, port uint16) (uint32, bool) {
	addr = addr.Unmap()
	var portOnly uint32
	var portHit bool
	for _, row := range rows {
		if row.Port != port || row.PID == 0 {
			continue
		}
		if row.HasAddr && row.Local.Unmap() == addr {
			return row.PID, true
		}
		if !row.HasAddr && !portHit {
			portOnly, portHit = row.PID, true
		}
	}
	return portOnly, portHit
}

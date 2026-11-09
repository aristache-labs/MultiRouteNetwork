package nics

import (
	"net"
	"testing"
)

func TestRenamedTunnelStaysVirtual(t *testing.T) {
	for _, name := range []string{"MultiRouteNetwork", "multiroutenetwork", "MULTIROUTENETWORK", "net-merge"} {
		if !virtualName.MatchString(name) {
			t.Fatalf("own tunnel %q must be excluded from routing", name)
		}
	}
	for _, name := range []string{"Ethernet", "Ethernet 4", "Wi-Fi"} {
		if virtualName.MatchString(name) {
			t.Fatalf("physical adapter %q must not be excluded", name)
		}
	}
}

func TestLinkCountersReadRealAdapter(t *testing.T) {
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		if _, _, ok := Octets(ifc.Index); !ok {
			t.Fatalf("could not read OS counters for %s (index %d)", ifc.Name, ifc.Index)
		}
		return
	}
	t.Skip("no up adapter")
}

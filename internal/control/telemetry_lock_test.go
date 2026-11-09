package control

import (
	"net"
	"MultiRouteNetwork/internal/rules"
	"testing"
	"time"
)

// Slow Windows enumeration must not suspend all TCP/UDP accounting callbacks.
func TestTelemetryEnumerationDoesNotBlockTransfer(t *testing.T) {
	for _, operation := range []string{"status", "sample"} {
		t.Run(operation, func(t *testing.T) {
			s := New("", nil)
			sw := &testSwitch{}
			sw.on.Store(true)
			s.sw = sw
			entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			s.interfaces = func() ([]net.Interface, error) { close(entered); <-release; return nil, nil }
			go func() {
				defer close(finished)
				if operation == "status" {
					s.Snapshot(nil, nil, true)
				} else {
					s.Sample(nil, nil, 250*time.Millisecond)
				}
			}()
			defer func() { close(release); <-finished }()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("enumeration not entered")
			}
			transferred := make(chan struct{})
			go func() { s.account("chrome.exe", "", "Ethernet", rules.Rule{}, true, 32768, false); close(transferred) }()
			select {
			case <-transferred:
			case <-time.After(time.Second):
				t.Fatal("telemetry blocked transfer accounting")
			}
		})
	}
}

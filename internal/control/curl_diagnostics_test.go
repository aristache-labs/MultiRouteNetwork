package control

import (
	"bufio"
	"encoding/json"
	"net-merge/internal/engine"
	"os"
	"path/filepath"
	"testing"
)

func TestCurlDiagnosticsFilterAndLifecycle(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, nil)
	s.EnableCurlDiagnostics()
	for _, h := range []engine.FlowHint{{Name: "chrome.exe", Known: true}, {Name: "curl.exe", Known: false}} {
		s.traceCurl(&h, "tcp", 10)
		if h.Trace != nil || h.TransferTrace != nil {
			t.Fatal("traced unrelated or unidentified process")
		}
	}
	h := engine.FlowHint{Name: "CURL.EXE", Known: true}
	s.traceCurl(&h, "tcp", 123)
	if h.Trace == nil || h.OnOpen == nil {
		t.Fatal("missing curl hooks")
	}
	h.Trace(engine.SelectionTrace{Selected: "Ethernet", Outcome: "selected"})
	closeFlow := h.OnOpen("Ethernet")
	h.TransferTrace(engine.TransferTrace{Phase: "progress", Adapter: "Ethernet", DownBytes: 100, UpBytes: 20, DownBps: 100})
	h.TransferTrace(engine.TransferTrace{Phase: "transfer-closed", Adapter: "Ethernet", DownBytes: 100, UpBytes: 20, ReadState: "eof", CloseReason: "local-close"})
	closeFlow()
	s.curlLog.close()
	// Emitting after shutdown is harmless and never sends to a closed channel.
	h.Trace(engine.SelectionTrace{})
	file, err := os.Open(filepath.Join(dir, "curl-routing.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var events []curlEvent
	for scanner.Scan() {
		var e curlEvent
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatal(err)
		}
		events = append(events, e)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{"prepared", "selection", "opened", "progress", "transfer-closed", "closed"}
	if len(events) != len(want) {
		t.Fatalf("events %v", events)
	}
	if events[4].Transfer == nil || events[4].Transfer.DownBytes != 100 || events[4].Transfer.ReadState != "eof" {
		t.Fatal("missing final transfer metrics")
	}
	for i, e := range events {
		if e.Event != want[i] || e.PID != 123 || e.Connection != 1 || e.Network != "tcp" {
			t.Fatalf("event %+v", e)
		}
	}
}

func TestCurlDiagnosticsRotateWithinBound(t *testing.T) {
	dir := t.TempDir()
	l := newCurlDiagnosticLog(dir)
	for i := 0; i < 12000; i++ {
		l.queue <- curlEvent{Event: "selection", Network: "tcp", PID: 123, Selection: &engine.SelectionTrace{Selected: "Wi-Fi", Outcome: "selected"}}
	}
	l.close()
	for _, name := range []string{"curl-routing.jsonl", "curl-routing.jsonl.1"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > curlLogMaxBytes {
			t.Fatalf("log exceeded bound: %d", info.Size())
		}
	}
}

package control

import (
	"encoding/json"
	"net-merge/internal/engine"
	"net-merge/internal/logx"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const curlLogMaxBytes = 1 << 20

type curlEvent struct {
	Time       time.Time              `json:"time"`
	Connection uint64                 `json:"connection"`
	PID        uint32                 `json:"pid"`
	Network    string                 `json:"network"`
	Event      string                 `json:"event"`
	Adapter    string                 `json:"adapter,omitempty"`
	Transfer   *engine.TransferTrace  `json:"transfer,omitempty"`
	Selection  *engine.SelectionTrace `json:"selection,omitempty"`
	Dropped    uint64                 `json:"droppedBefore,omitempty"`
}
type curlDiagnosticLog struct {
	mu       sync.RWMutex
	closed   bool
	queue    chan curlEvent
	done     chan struct{}
	sequence atomic.Uint64
	dropped  atomic.Uint64
}

func newCurlDiagnosticLog(dir string) *curlDiagnosticLog {
	l := &curlDiagnosticLog{queue: make(chan curlEvent, 256), done: make(chan struct{})}
	go l.write(filepath.Join(dir, "curl-routing.jsonl"))
	return l
}
func (l *curlDiagnosticLog) emit(e curlEvent) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.closed {
		return
	}
	e.Time = time.Now()
	select {
	case l.queue <- e:
	default:
		l.dropped.Add(1)
	}
}
func (l *curlDiagnosticLog) close() {
	l.mu.Lock()
	if !l.closed {
		l.closed = true
		close(l.queue)
	}
	l.mu.Unlock()
	<-l.done
}
func (l *curlDiagnosticLog) write(path string) {
	defer close(l.done)
	var file *os.File
	defer func() {
		if file != nil {
			file.Close()
		}
	}()
	var size int64
	for e := range l.queue {
		e.Dropped = l.dropped.Swap(0)
		data, err := json.Marshal(e)
		if err != nil {
			continue
		}
		data = append(data, '\n')
		if file == nil {
			file, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				logx.Printf("warning: curl diagnostic log: %v\n", err)
				continue
			}
			if info, err := file.Stat(); err == nil {
				size = info.Size()
			}
		}
		if size+int64(len(data)) > curlLogMaxBytes {
			file.Close()
			file = nil
			_ = os.Remove(path + ".1")
			if err = os.Rename(path, path+".1"); err != nil {
				logx.Printf("warning: curl diagnostic rotation: %v\n", err)
				continue
			}
			file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			if err != nil {
				continue
			}
			size = 0
		}
		n, err := file.Write(data)
		size += int64(n)
		if err != nil {
			logx.Printf("warning: curl diagnostic write: %v\n", err)
			file.Close()
			file = nil
		}
	}
}

// EnableCurlDiagnostics is called at engine initialization, before Prepare is exposed.
// Only identified curl.exe processes are traced. No URLs, IPs or payloads are logged.
func (s *Service) EnableCurlDiagnostics() { s.curlLog = newCurlDiagnosticLog(s.dir) }

func (s *Service) traceCurl(h *engine.FlowHint, network string, pid uint32) {
	if s.curlLog != nil && h.Known && strings.EqualFold(h.Name, "curl.exe") {
		logger := s.curlLog
		id := logger.sequence.Add(1)
		emit := func(e curlEvent) { e.Connection = id; e.PID = pid; e.Network = network; logger.emit(e) }
		emit(curlEvent{Event: "prepared"})
		h.TransferTrace = func(v engine.TransferTrace) { emit(curlEvent{Event: v.Phase, Adapter: v.Adapter, Transfer: &v}) }
		h.Trace = func(trace engine.SelectionTrace) { emit(curlEvent{Event: "selection", Selection: &trace}) }
		h.OnOpen = func(adapter string) func() {
			done := s.trackAdapter(h.Name, adapter)
			emit(curlEvent{Event: "opened", Adapter: adapter})
			return func() { done(); emit(curlEvent{Event: "closed", Adapter: adapter}) }
		}
	}
}

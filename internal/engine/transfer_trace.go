package engine

import (
	"errors"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

type TransferTrace struct {
	Phase          string  `json:"phase"`
	Adapter        string  `json:"adapter"`
	DurationMs     int64   `json:"durationMs"`
	DownBytes      int64   `json:"downBytes"`
	UpBytes        int64   `json:"upBytes"`
	DownBps        float64 `json:"downBps"` // bytes/s over the sampling interval
	UpBps          float64 `json:"upBps"`
	AverageDownBps float64 `json:"averageDownBps"`
	AverageUpBps   float64 `json:"averageUpBps"`
	ReadState      string  `json:"readState"`
	WriteState     string  `json:"writeState"`
	CloseReason    string  `json:"closeReason,omitempty"`
}
type transferMeter struct {
	mu               sync.Mutex
	ops              sync.WaitGroup
	closing          bool
	done             chan struct{}
	workerDone       chan struct{}
	emit             func(TransferTrace)
	start, last      time.Time
	value            TransferTrace
	lastDown, lastUp int64
}

func newTransferMeter(adapter string, emit func(TransferTrace)) *transferMeter {
	if emit == nil {
		return nil
	}
	now := time.Now()
	m := &transferMeter{done: make(chan struct{}), workerDone: make(chan struct{}), emit: emit, start: now, last: now, value: TransferTrace{Adapter: adapter, ReadState: "no-terminal-observed", WriteState: "no-terminal-observed"}}
	go func() {
		defer close(m.workerDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-m.done:
				return
			case <-ticker.C:
				m.sample("progress", "")
			}
		}
	}()
	return m
}
func (m *transferMeter) begin() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closing {
		return false
	}
	m.ops.Add(1)
	return true
}
func (m *transferMeter) end(down bool, n int, err error) {
	m.mu.Lock()
	if down {
		m.value.DownBytes += int64(n)
		if err != nil {
			m.value.ReadState = transferError(err)
		}
	} else {
		m.value.UpBytes += int64(n)
		if err != nil {
			m.value.WriteState = transferError(err)
		}
	}
	m.mu.Unlock()
	m.ops.Done()
}
func (m *transferMeter) sample(phase, reason string) {
	m.mu.Lock()
	now := time.Now()
	seconds := now.Sub(m.last).Seconds()
	elapsed := now.Sub(m.start).Seconds()
	v := m.value
	v.Phase = phase
	v.CloseReason = reason
	v.DurationMs = now.Sub(m.start).Milliseconds()
	if seconds > 0 {
		v.DownBps = float64(v.DownBytes-m.lastDown) / seconds
		v.UpBps = float64(v.UpBytes-m.lastUp) / seconds
	}
	if elapsed > 0 {
		v.AverageDownBps = float64(v.DownBytes) / elapsed
		v.AverageUpBps = float64(v.UpBytes) / elapsed
	}
	m.last = now
	m.lastDown = v.DownBytes
	m.lastUp = v.UpBytes
	m.mu.Unlock()
	m.emit(v)
}
func (m *transferMeter) finish(reason string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.closing = true
	close(m.done)
	m.mu.Unlock()
	<-m.workerDone
	m.ops.Wait()
	m.sample("transfer-closed", reason)
}
func transferError(err error) string {
	var timeout net.Error
	switch {
	case errors.Is(err, io.EOF):
		return "eof"
	case errors.Is(err, net.ErrClosed):
		return "closed"
	case errors.As(err, &timeout) && timeout.Timeout():
		return "timeout"
	case errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.Errno(10054)):
		return "reset"
	case errors.Is(err, syscall.EPIPE):
		return "broken-pipe"
	default:
		return "error"
	}
}

// Count only actual socket I/O, not time spent sleeping in the existing limiter.
// Closing diagnostics must not wait for a pacing sleep to finish.
type transferWriter struct {
	io.Writer
	meter *transferMeter
}

func (w transferWriter) Write(p []byte) (int, error) {
	traced := w.meter.begin()
	n, err := w.Writer.Write(p)
	if traced {
		w.meter.end(false, n, err)
	}
	return n, err
}

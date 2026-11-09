package tun

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/xjasonlyu/tun2socks/v2/core/device"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// packetDevice replaces the IO-based link endpoint, not the underlying native
// device or its packet format. The raw endpoint is never attached to the stack.
type packetDevice struct {
	*channel.Endpoint
	raw             device.Device
	rw              io.ReadWriter
	offset          int
	mtu             int
	ctx             context.Context
	cancel          context.CancelFunc
	lifecycle       sync.Mutex
	started, closed bool
	wg              sync.WaitGroup
}

func newPacketDevice(raw device.Device, rw io.ReadWriter, offset int) (*packetDevice, error) {
	if raw == nil || rw == nil || raw.MTU() == 0 || offset < 0 {
		return nil, errors.New("invalid packet device")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &packetDevice{Endpoint: channel.New(1<<10, raw.MTU(), ""), raw: raw, rw: rw, offset: offset, mtu: int(raw.MTU()), ctx: ctx, cancel: cancel}, nil
}
func (d *packetDevice) Name() string { return d.raw.Name() }
func (d *packetDevice) Type() string { return d.raw.Type() }
func (d *packetDevice) Attach(dispatcher stack.NetworkDispatcher) {
	d.lifecycle.Lock()
	defer d.lifecycle.Unlock()
	if d.closed {
		return
	}
	d.Endpoint.Attach(dispatcher)
	if d.started || dispatcher == nil {
		return
	}
	d.started = true
	d.wg.Add(2)
	go func() { defer d.wg.Done(); d.dispatchLoop() }()
	go func() { defer d.wg.Done(); d.outboundLoop() }()
}
func (d *packetDevice) Close() {
	d.lifecycle.Lock()
	if !d.closed {
		d.closed = true
		d.cancel()
		d.raw.Close()
		d.Endpoint.Close()
	}
	d.lifecycle.Unlock()
	d.wg.Wait()
}
func (d *packetDevice) Wait() { d.wg.Wait() }

func (d *packetDevice) dispatchLoop() {
	defer d.cancel()
	scratch := make([]byte, d.offset+d.mtu)
	for {
		n, err := d.rw.Read(scratch)
		if err != nil {
			return
		}
		if n <= 0 || n > d.mtu || !d.IsAttached() {
			continue
		}
		payload := scratch[d.offset : d.offset+n]
		var protocol tcpip.NetworkProtocolNumber
		switch header.IPVersion(payload) {
		case header.IPv4Version:
			protocol = header.IPv4ProtocolNumber
		case header.IPv6Version:
			protocol = header.IPv6ProtocolNumber
		default:
			continue
		}
		// MakeWithData copies into reference-counted, pooled gVisor storage. The
		// scratch slice is never borrowed by the stack, even if it retains pkt.
		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: buffer.MakeWithData(payload)})
		d.InjectInbound(protocol, pkt)
		pkt.DecRef()
	}
}

func (d *packetDevice) outboundLoop() {
	scratch := make([]byte, d.offset+d.mtu)
	for {
		pkt := d.ReadContext(d.ctx)
		if pkt == nil {
			return
		}
		needed := d.offset + pkt.Size()
		if needed > cap(scratch) {
			scratch = make([]byte, needed)
		}
		packet := scratch[:needed]
		clear(packet[:d.offset])
		copyPacketBytes(packet[d.offset:], pkt)
		// NativeTun.Write returns a packet count, not a byte count. Preserve the
		// original IO endpoint's write contract and one-write-per-packet framing.
		_, _ = d.rw.Write(packet)
		pkt.DecRef()
	}
}

// Walk borrowed views while pkt is still owned. Unlike ToBuffer().Flatten(),
// this does not allocate packet-sized storage or clone buffer/view references.
func copyPacketBytes(dst []byte, pkt *stack.PacketBuffer) int {
	views, skip := pkt.AsViewList()
	n := 0
	for v := views.Front(); v != nil; v = v.Next() {
		data := v.AsSlice()
		if skip >= len(data) {
			skip -= len(data)
			continue
		}
		data = data[skip:]
		skip = 0
		n += copy(dst[n:], data)
	}
	return n
}

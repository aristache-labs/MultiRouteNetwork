package tun

import (
	"bytes"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type memoryPacketDevice struct {
	*channel.Endpoint
	input  chan []byte
	output chan []byte
	done   chan struct{}
	once   sync.Once
	closes atomic.Int32
	offset int
}

func newMemoryPacketDevice(mtu uint32, offset int) *memoryPacketDevice {
	return &memoryPacketDevice{Endpoint: channel.New(1, mtu, ""), input: make(chan []byte, 16), output: make(chan []byte, 16), done: make(chan struct{}), offset: offset}
}
func (d *memoryPacketDevice) Name() string { return "memory" }
func (d *memoryPacketDevice) Type() string { return "test" }
func (d *memoryPacketDevice) Read(p []byte) (int, error) {
	select {
	case v := <-d.input:
		copy(p[d.offset:], v)
		return len(v), nil
	case <-d.done:
		return 0, net.ErrClosed
	}
}
func (d *memoryPacketDevice) Write(p []byte) (int, error) {
	v := append([]byte(nil), p...)
	select {
	case d.output <- v:
		return 1, nil
	case <-d.done:
		return 0, net.ErrClosed
	}
}
func (d *memoryPacketDevice) Close() {
	d.once.Do(func() { d.closes.Add(1); close(d.done); d.Endpoint.Close() })
}

type retainedPacket struct {
	protocol tcpip.NetworkProtocolNumber
	pkt      *stack.PacketBuffer
}
type retainingDispatcher struct{ packets chan retainedPacket }

func (d *retainingDispatcher) DeliverNetworkPacket(p tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) {
	d.packets <- retainedPacket{p, pkt.IncRef()}
}
func (d *retainingDispatcher) DeliverLinkPacket(tcpip.NetworkProtocolNumber, *stack.PacketBuffer) {}

func TestPacketDeviceInboundScratchDoesNotAliasRetainedPackets(t *testing.T) {
	raw := newMemoryPacketDevice(9000, 0)
	dev, err := newPacketDevice(raw, raw, 0)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &retainingDispatcher{packets: make(chan retainedPacket, 4)}
	dev.Attach(dispatcher)
	defer dev.Close()
	originals := [][]byte{append([]byte{0x45}, bytes.Repeat([]byte{1}, 1399)...), append([]byte{0x60}, bytes.Repeat([]byte{2}, 79)...)}
	for _, p := range originals {
		raw.input <- p
	}
	held := []retainedPacket{}
	for range originals {
		select {
		case p := <-dispatcher.packets:
			held = append(held, p)
		case <-time.After(time.Second):
			t.Fatal("missing inbound packet")
		}
	}
	dev.Close() // Retained stack references must survive device close too.
	for i, p := range held {
		got := make([]byte, p.pkt.Size())
		copyPacketBytes(got, p.pkt)
		p.pkt.DecRef()
		if !bytes.Equal(got, originals[i]) {
			t.Fatalf("packet %d changed after scratch reuse", i)
		}
	}
	if held[0].protocol != 0x0800 || held[1].protocol != 0x86dd {
		t.Fatal("lost IPv4/IPv6 protocol")
	}
	if raw.closes.Load() != 1 {
		t.Fatal("native device closed more than once")
	}
}

func makeSegmentedPacket(size int, onRelease func()) *stack.PacketBuffer {
	var payload buffer.Buffer
	for i := 0; i < size; {
		n := min(size-i, 137)
		payload.Append(buffer.NewViewWithData(bytes.Repeat([]byte{byte(i % 251)}, n)))
		i += n
	}
	pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{ReserveHeaderBytes: 96, Payload: payload, OnRelease: onRelease})
	copy(pkt.TransportHeader().Push(8), []byte("UDP-test"))
	copy(pkt.NetworkHeader().Push(20), []byte("IPv4-header-test-1234"))
	return pkt
}

func TestPacketDeviceOutboundViewsOffsetsGrowthAndRelease(t *testing.T) {
	for _, offset := range []int{0, 4} {
		raw := newMemoryPacketDevice(1500, offset)
		dev, err := newPacketDevice(raw, raw, offset)
		if err != nil {
			t.Fatal(err)
		}
		dev.Attach(&retainingDispatcher{packets: make(chan retainedPacket, 1)})
		var released atomic.Int32
		for _, size := range []int{40, 1400, 18000, 70} {
			pkt := makeSegmentedPacket(size, func() { released.Add(1) })
			buf := pkt.ToBuffer()
			want := append(make([]byte, offset), buf.Flatten()...)
			buf.Release()
			var list stack.PacketBufferList
			list.PushBack(pkt)
			if n, err := dev.WritePackets(list); err != nil || n != 1 {
				t.Fatalf("enqueue %d %v", n, err)
			}
			pkt.DecRef()
			select {
			case got := <-raw.output:
				if !bytes.Equal(want, got) {
					t.Fatalf("framing mismatch offset=%d size=%d", offset, size)
				}
			case <-time.After(time.Second):
				t.Fatal("missing output packet")
			}
		}
		dev.Close()
		dev.Wait()
		if released.Load() != 4 {
			t.Fatalf("packet references leaked: %d", released.Load())
		}
	}
}

func TestPacketDeviceCloseUnblocksIOAndRejectsReattach(t *testing.T) {
	raw := newMemoryPacketDevice(1500, 0)
	raw.output = make(chan []byte) // Block native write.
	dev, _ := newPacketDevice(raw, raw, 0)
	dispatcher := &retainingDispatcher{packets: make(chan retainedPacket, 1)}
	dev.Attach(dispatcher)
	var released atomic.Int32
	for i := 0; i < 8; i++ {
		pkt := makeSegmentedPacket(100, func() { released.Add(1) })
		var list stack.PacketBufferList
		list.PushBack(pkt)
		dev.WritePackets(list)
		pkt.DecRef()
	}
	done := make(chan struct{})
	go func() { dev.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("blocked IO prevented close")
	}
	dev.Attach(dispatcher)
	dev.Close()
	if released.Load() != 8 || raw.closes.Load() != 1 {
		t.Fatalf("close leaked refs or repeated native close: %d/%d", released.Load(), raw.closes.Load())
	}
}

func TestPacketDeviceDropsInvalidInput(t *testing.T) {
	raw := newMemoryPacketDevice(100, 0)
	dev, _ := newPacketDevice(raw, raw, 0)
	dispatcher := &retainingDispatcher{packets: make(chan retainedPacket, 4)}
	dev.Attach(dispatcher)
	defer dev.Close()
	for _, p := range [][]byte{nil, {0xf0}, bytes.Repeat([]byte{0x45}, 101), {0x60, 1, 2}} {
		raw.input <- p
	}
	select {
	case p := <-dispatcher.packets:
		if p.protocol != 0x86dd {
			t.Fatal("accepted malformed input")
		}
		p.pkt.DecRef()
	case <-time.After(time.Second):
		t.Fatal("valid packet lost")
	}
	select {
	case p := <-dispatcher.packets:
		p.pkt.DecRef()
		t.Fatal("invalid packet delivered")
	default:
	}
}

var benchmarkMTU = 9000
var packetBenchmarkSink byte

func BenchmarkPacketOutboundFlatten(b *testing.B) {
	pkt := makeSegmentedPacket(8972, nil)
	defer pkt.DecRef()
	b.ReportAllocs()
	b.SetBytes(int64(pkt.Size()))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pkt.ToBuffer()
		flat := buf.Flatten()
		packetBenchmarkSink = flat[0]
		buf.Release()
	}
}
func BenchmarkPacketOutboundReuse(b *testing.B) {
	pkt := makeSegmentedPacket(8972, nil)
	defer pkt.DecRef()
	scratch := make([]byte, pkt.Size())
	b.ReportAllocs()
	b.SetBytes(int64(pkt.Size()))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copyPacketBytes(scratch, pkt)
		packetBenchmarkSink = scratch[0]
	}
}
func BenchmarkPacketInboundAllocate(b *testing.B) {
	payload := bytes.Repeat([]byte{0x45}, 40)
	b.ReportAllocs()
	b.SetBytes(40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scratch := make([]byte, benchmarkMTU)
		copy(scratch, payload)
		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: buffer.MakeWithData(scratch[:40])})
		pkt.DecRef()
	}
}
func BenchmarkPacketInboundReuse(b *testing.B) {
	payload := bytes.Repeat([]byte{0x45}, 40)
	scratch := make([]byte, benchmarkMTU)
	b.ReportAllocs()
	b.SetBytes(40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(scratch, payload)
		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: buffer.MakeWithData(scratch[:40])})
		pkt.DecRef()
	}
}

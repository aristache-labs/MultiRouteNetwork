package control

import (
	"encoding/csv"
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"net/netip"

	"MultiRouteNetwork/internal/engine"
	"MultiRouteNetwork/internal/nics"
	"MultiRouteNetwork/internal/procid"
	"MultiRouteNetwork/internal/rules"
	"MultiRouteNetwork/internal/store"
)

const (
	graphLen = 240
	warnAt   = 0.8
)

// Switch is the TUN the window and the tray both turn on and off.
type Switch interface {
	On() error
	Off()
	IsOn() bool
}

// AdapterCard is one network adapter plus a minute of throughput.
type AdapterCard struct {
	Name         string    `json:"name"`
	IP           string    `json:"ip"`
	IP6          string    `json:"ip6"`
	State        string    `json:"state"`
	Excluded     bool      `json:"excluded"`
	Reason       string    `json:"reason,omitempty"`
	DownBps      float64   `json:"downBps"`
	UpBps        float64   `json:"upBps"`
	DownBytes    int64     `json:"downBytes"`
	UpBytes      int64     `json:"upBytes"`
	LatencyMs    int64     `json:"latencyMs"`
	Down         []float64 `json:"down"`
	Up           []float64 `json:"up"`
	Default      bool      `json:"default"`
	Routable     bool      `json:"routable"`
	LinkDownMbps float64   `json:"linkDownMbps"`
	LinkUpMbps   float64   `json:"linkUpMbps"`
	TodayDown    int64     `json:"todayDown"`
	TodayUp      int64     `json:"todayUp"`
	MonthDown    int64     `json:"monthDown"`
	MonthUp      int64     `json:"monthUp"`
}

// AppRow is one program in the dashboard table.
type AppRow struct {
	Running      bool `json:"running"`
	RunningKnown bool `json:"runningKnown"`

	Exe       string    `json:"exe"`
	Path      string    `json:"path"`
	DownBps   float64   `json:"downBps"`
	UpBps     float64   `json:"upBps"`
	DownBytes int64     `json:"downBytes"`
	UpBytes   int64     `json:"upBytes"`
	Via       string    `json:"via"`
	Warn      bool      `json:"warn"`
	Capped    bool      `json:"capped"`
	Down      []float64 `json:"down"`
	Up        []float64 `json:"up"`
}

// Status is the snapshot pushed about once a second.
type Status struct {
	Running  bool           `json:"running"`
	Starting bool           `json:"starting,omitempty"`
	Error    string         `json:"error,omitempty"`
	Theme    string         `json:"theme"`
	Default  string         `json:"defaultAdapter"`
	DownBps  float64        `json:"downBps"`
	UpBps    float64        `json:"upBps"`
	Down     []float64      `json:"down"`
	Up       []float64      `json:"up"`
	Adapters []AdapterCard  `json:"adapters"`
	Apps     []AppRow       `json:"apps"`
	Rules    []rules.Rule   `json:"rules"`
	Settings store.Settings `json:"settings"`
	Updated  time.Time      `json:"updated"`
}

// Service is the shared controller for the tray, the CLI and the desktop window.
type Service struct {
	curlLog        *curlDiagnosticLog
	adapterBuckets map[string]*bucket

	sampleSeconds float64

	trafficPending map[string]map[string]int64
	trafficRates   map[string]map[string]int64

	openAdapters map[string]map[string]int

	adapterCache *adapterIdentityCache
	octets       func(int) (uint64, uint64, bool)
	interfaces   func() ([]net.Interface, error)
	mu           sync.Mutex
	configMu     sync.Mutex
	sw           Switch
	sched        *engine.Scheduler
	db           *store.Store
	dir          string
	rules        atomic.Pointer[[]rules.Rule]
	settings     atomic.Pointer[store.Settings]
	hold         rules.Hold
	lastErr      string
	graphs       map[string]*series
	apps         map[string]*appMeter
	lastSnap     []engine.SlotStats
	lastSkip     []nics.Skip
	linkPrev     map[string]linkSample
	uses         map[string]*adapterUse
	total        series
	paused       atomic.Bool
	recording    bool
}

type series struct {
	down, up         []float64
	lastDown, lastUp float64
}

type adapterUse struct {
	dayKey, monthKey                       string
	todayDown, todayUp, monthDown, monthUp int64
	loaded                                 bool
}

type linkSample struct {
	in, out         uint64
	rateIn, rateOut float64
	primed          bool
	index           int
	hardware        string
}

type appMeter struct {
	path                   string
	down, up               int64
	flushedDown, flushedUp int64
	period                 int64
	periodKey              string
	periodLoaded           bool
	lastDown, lastUp       int64
	rateDown, rateUp       float64
	via                    string
	graph                  []float64
	graphUp                []float64
	warn, capped           bool
	buckets                map[string]*bucket
}

func New(dir string, db *store.Store) *Service {
	s := &Service{octets: nics.Octets, interfaces: net.Interfaces, dir: dir, db: db, graphs: map[string]*series{}, apps: map[string]*appMeter{}, linkPrev: map[string]linkSample{}, uses: map[string]*adapterUse{}}
	s.adapterCache = newAdapterIdentityCache(func() ([]net.Interface, error) { return s.interfaces() })
	var list []rules.Rule
	st := store.DefaultSettings()
	if db != nil {
		list, _ = db.Rules()
		st, _ = db.LoadSettings()
	}
	s.rules.Store(&list)
	s.settings.Store(&st)
	s.total.down = make([]float64, graphLen)
	s.total.up = make([]float64, graphLen)
	return s
}

// Attach binds a live engine. Policy runs without taking the service lock.
func (s *Service) Attach(sched *engine.Scheduler, sw Switch) {
	s.mu.Lock()
	s.sched, s.sw = sched, sw
	s.recording = sw != nil && sw.IsOn()
	s.mu.Unlock()
	sched.SetPolicy(s.decide)
}

func (s *Service) SetError(err string) {
	s.mu.Lock()
	s.lastErr = err
	s.mu.Unlock()
}

func (s *Service) decide(hint engine.FlowHint, views []engine.AdapterView) engine.Decision {
	if hint.Blocked {
		return engine.Decision{Block: true}
	}
	ads := make([]rules.Adapter, len(views))
	for i, v := range views {
		ads[i] = rules.Adapter{Name: v.Name, Up: v.Up, Latency: v.Latency, Default: v.Name == s.Settings().DefaultAdapter}
	}
	list := s.ruleList()
	s.mu.Lock()
	res := rules.Choose(list, rules.Proc{Name: hint.Name, Path: hint.Path, Known: hint.Known}, ads, &s.hold, time.Now())
	s.mu.Unlock()
	if res.Block {
		return engine.Decision{Block: true}
	}
	if res.Balance {
		return engine.Decision{}
	}
	if res.Pin != "" {
		return engine.Decision{Pin: res.Pin}
	}
	return engine.Decision{Only: res.Only}
}

// Settings returns the saved desktop settings.
func (s *Service) Settings() store.Settings {
	if p := s.settings.Load(); p != nil {
		return *p
	}
	return store.DefaultSettings()
}

func (s *Service) ruleList() []rules.Rule {
	p := s.rules.Load()
	if p == nil {
		return nil
	}
	return *p
}

func (s *Service) Prepare(network string, src, dst string, srcPort, dstPort int) engine.FlowHint {
	_ = dst
	_ = dstPort
	addr, err := netip.ParseAddr(src)
	h := engine.FlowHint{}
	var pid uint32
	if err == nil {
		p := procid.Lookup(network, addr, uint16(srcPort))
		h.Name, h.Path, h.Known = p.Name, p.Path, p.Known
		pid = p.PID
	}
	if !h.Known {
		h.Name = "Unknown"
	}
	rule, _ := matchRule(s.ruleList(), h)
	h.DNS = dnsHost(rule)
	if s.capped(rule, h.Name) {
		h.Blocked = true
		s.markCapped(h.Name, h.Path)
	}
	name, path := h.Name, h.Path
	h.OnOpen = func(adapter string) func() { return s.trackAdapter(name, adapter) }
	s.traceCurl(&h, network, pid)
	h.OnBytes = func(adapter string, down bool, n int, udp bool) {
		s.account(name, path, adapter, rule, down, n, udp)
	}
	h.Stop = func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		m := s.apps[name]
		return m != nil && m.capped
	}
	return h
}

func (s *Service) account(exe, path, adapter string, rule rules.Rule, down bool, n int, udp bool) {
	if s.paused.Load() || !s.isOn() {
		return
	}
	if exe == "" {
		exe = "Unknown"
	}
	_ = udp // per-app limits count small datagrams too; the adapter bucket still exempts them
	s.mu.Lock()
	if s.paused.Load() || s.sw == nil || !s.sw.IsOn() {
		s.mu.Unlock()
		return
	}
	m := s.apps[exe]
	if m == nil {
		m = &appMeter{graph: make([]float64, graphLen), graphUp: make([]float64, graphLen), buckets: map[string]*bucket{}}
		s.apps[exe] = m
	}
	m.path = path
	if s.trafficPending == nil {
		s.trafficPending = map[string]map[string]int64{}
	}
	key := strings.ToLower(exe)
	if s.trafficPending[key] == nil {
		s.trafficPending[key] = map[string]int64{}
	}
	s.trafficPending[key][adapter] += int64(n)
	if adapter != "" {
		m.via = adapter
	}
	if down {
		m.down += int64(n)
	} else {
		m.up += int64(n)
	}
	if rule.CapGB > 0 {
		key := rules.CapKey(exe, rule.CapPeriod, time.Now())
		if !m.periodLoaded || m.periodKey != key {
			m.periodKey = key
			m.period = 0
			m.periodLoaded = true
			m.capped = false
			if s.db != nil {
				if already, err := s.db.PeriodBytes(exe, periodStart(rule.CapPeriod, time.Now())); err == nil {
					m.period = already
				}
			}
		}
		m.period += int64(n)
		limit := rule.CapGB * 1e9
		m.warn = float64(m.period) >= limit*warnAt
		m.capped = float64(m.period) >= limit
	}
	wait := s.paceLocked(m, rule, adapter, down, n)
	s.mu.Unlock()
	if wait > 0 {
		time.Sleep(wait)
	}
}

func (s *Service) paceLocked(m *appMeter, rule rules.Rule, adapter string, down bool, n int) time.Duration {
	var wait time.Duration
	take := func(key string, mbps float64) {
		if mbps <= 0 {
			return
		}
		b := m.buckets[key]
		if b == nil || b.mbps != mbps {
			b = newBucket(mbps)
			m.buckets[key] = b
		}
		if d := b.take(n); d > wait {
			wait = d
		}
	}
	if down {
		take("app-down", rule.DownMbps)
		take("ad-down:"+adapter, rule.AdapterDown[adapter])
	} else {
		take("app-up", rule.UpMbps)
		take("aa-up:"+adapter, rule.AdapterUp[adapter])
	}
	if st := s.settings.Load(); st != nil {
		mbps := st.AdapterUp[adapter]
		key := "up:" + adapter
		if down {
			mbps = st.AdapterDown[adapter]
			key = "down:" + adapter
		}
		if mbps > 0 {
			if s.adapterBuckets == nil {
				s.adapterBuckets = map[string]*bucket{}
			}
			b := s.adapterBuckets[key]
			if b == nil || b.mbps != mbps {
				b = newBucket(mbps)
				s.adapterBuckets[key] = b
			}
			if d := b.take(n); d > wait {
				wait = d
			}
		}
		if down {
			take("global-down", st.DownMbps)
		} else {
			take("global-up", st.UpMbps)
		}
	}
	return wait
}

func (s *Service) capped(rule rules.Rule, exe string) bool {
	if rule.CapGB <= 0 || s.db == nil {
		return false
	}
	from := periodStart(rule.CapPeriod, time.Now())
	n, err := s.db.PeriodBytes(exe, from)
	if err != nil {
		return false
	}
	return float64(n) >= rule.CapGB*1e9
}

func (s *Service) markCapped(exe, path string) {
	s.mu.Lock()
	m := s.apps[exe]
	if m == nil {
		m = &appMeter{path: path, graph: make([]float64, graphLen), graphUp: make([]float64, graphLen)}
		s.apps[exe] = m
	}
	m.capped = true
	s.mu.Unlock()
}

// Tick records one second of adapter throughput. Call it from the engine loop.
func (s *Service) Tick(snap []engine.SlotStats, skipped []nics.Skip) {
	s.tick(snap, skipped, time.Second, false)
}

// Sample reads OS counters at the telemetry cadence without ticking the scheduler.
func (s *Service) Sample(snap []engine.SlotStats, skipped []nics.Skip, dt time.Duration) {
	s.tick(snap, skipped, dt, true)
}
func (s *Service) tick(snap []engine.SlotStats, skipped []nics.Skip, dt time.Duration, physical bool) {
	if dt <= 0 {
		return
	}
	running := s.isOn() && !s.paused.Load()
	// Windows adapter enumeration can take hundreds of milliseconds. Never hold
	// the accounting mutex while calling it: every relayed buffer needs that lock.
	readings := map[string]linkReading{}
	if running {
		byName := s.adapterCache.get()
		wanted := map[string]bool{}
		if physical {
			for _, sl := range snap {
				if sl.Present {
					wanted[sl.Name] = true
				}
			}
		}
		for _, sk := range skipped {
			if !hideAdapter(sk.Name, sk.Reason) {
				wanted[sk.Name] = true
			}
		}
		for name := range wanted {
			if ifc, found := byName[name]; found {
				in, out, ok := s.octets(ifc.Index)
				readings[name] = linkReading{in: in, out: out, ok: ok, index: ifc.Index, hardware: ifc.HardwareAddr.String()}
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSnap = append([]engine.SlotStats(nil), snap...)
	s.lastSkip = append([]nics.Skip(nil), skipped...)
	if !running || s.paused.Load() || s.sw == nil || !s.sw.IsOn() {
		s.recording = false
		return
	}
	s.sampleSeconds = dt.Seconds()
	s.trafficRates, s.trafficPending = s.trafficPending, nil
	// Discard the first OS-counter interval after resume: it may include stopped traffic.
	if !s.recording {
		s.linkPrev = map[string]linkSample{}
		if physical {
			for _, sl := range snap {
				s.sampleLink(sl.Name, readings[sl.Name])
			}
		}
		for _, sk := range skipped {
			if !hideAdapter(sk.Name, sk.Reason) {
				s.sampleLink(sk.Name, readings[sk.Name])
			}
		}
		s.recording = true
		return
	}
	var down, up float64
	seen := map[string]bool{}
	for _, sl := range snap {
		if !sl.Present {
			continue
		}
		seen[sl.Name] = true
		if physical {
			sl.LinkIn, sl.LinkOut = s.sampleLink(sl.Name, readings[sl.Name])
			for i := range s.lastSnap {
				if s.lastSnap[i].Name == sl.Name {
					s.lastSnap[i].LinkIn, s.lastSnap[i].LinkOut = sl.LinkIn, sl.LinkOut
				}
			}
		}
		g := s.graphs[sl.Name]
		if g == nil {
			g = &series{down: make([]float64, graphLen), up: make([]float64, graphLen)}
			s.graphs[sl.Name] = g
		}
		push(g.down, sl.LinkIn)
		push(g.up, sl.LinkOut)
		s.addAdapterBytes(sl.Name, int64(sl.LinkIn*dt.Seconds()), int64(sl.LinkOut*dt.Seconds()))
		down += sl.LinkIn
		up += sl.LinkOut
	}
	for _, sk := range skipped {
		seen[sk.Name] = true
		if hideAdapter(sk.Name, sk.Reason) {
			continue
		}
		rateIn, rateOut := s.sampleLink(sk.Name, readings[sk.Name])
		g := s.graphs[sk.Name]
		if g == nil {
			g = &series{down: make([]float64, graphLen), up: make([]float64, graphLen)}
			s.graphs[sk.Name] = g
		}
		push(g.down, rateIn)
		push(g.up, rateOut)
		s.addAdapterBytes(sk.Name, int64(rateIn*dt.Seconds()), int64(rateOut*dt.Seconds()))
	}
	push(s.total.down, down)
	push(s.total.up, up)
	for _, m := range s.apps {
		d := float64(m.down - m.lastDown)
		m.rateDown, m.rateUp = d/dt.Seconds(), float64(m.up-m.lastUp)/dt.Seconds()
		m.lastDown, m.lastUp = m.down, m.up
		push(m.graph, m.rateDown)
		if m.graphUp == nil {
			m.graphUp = make([]float64, graphLen)
		}
		push(m.graphUp, m.rateUp)
	}
	_ = seen
}

func (s *Service) Snapshot(snap []engine.SlotStats, skipped []nics.Skip, running bool) Status {
	processes, processErr := procid.Running()
	s.mu.Lock()
	st := store.DefaultSettings()
	if p := s.settings.Load(); p != nil {
		st = *p
	}
	out := Status{
		Running: running, Error: s.lastErr, Theme: st.Theme, Default: st.DefaultAdapter,
		Down: append([]float64(nil), s.total.down...), Up: append([]float64(nil), s.total.up...),
		Rules: s.ruleList(), Settings: st, Updated: time.Now(),
	}
	if snap == nil {
		snap, skipped = s.lastSnap, s.lastSkip
	}
	for _, sl := range snap {
		if !sl.Present {
			continue
		}
		card := AdapterCard{
			Name: sl.Name, IP: sl.IP, IP6: sl.IP6, State: onlineState(sl.State),
			DownBps: sl.LinkIn, UpBps: sl.LinkOut, LatencyMs: sl.LatencyMs, Routable: true,
		}
		if g := s.graphs[sl.Name]; g != nil {
			card.Down = append([]float64(nil), g.down...)
			card.Up = append([]float64(nil), g.up...)
		}
		card.Default = sl.Name == st.DefaultAdapter
		s.fillUse(&card)
		out.DownBps += sl.LinkIn
		out.UpBps += sl.LinkOut
		out.Adapters = append(out.Adapters, card)
	}
	for _, sk := range skipped {
		if hideAdapter(sk.Name, sk.Reason) {
			continue
		}
		state := "online"
		if vpnAdapter(sk.Name, sk.Reason) {
			state = "VPN"
		}
		card := AdapterCard{Name: sk.Name, State: state, Reason: sk.Reason, Routable: false}
		if prev, ok := s.linkPrev[sk.Name]; ok {
			card.DownBps, card.UpBps = prev.rateIn, prev.rateOut
		}
		if g := s.graphs[sk.Name]; g != nil {
			card.Down = append([]float64(nil), g.down...)
			card.Up = append([]float64(nil), g.up...)
		}
		s.fillUse(&card)
		out.Adapters = append(out.Adapters, card)
	}
	for _, rule := range s.ruleList() {
		row := AppRow{Exe: rule.Exe, Path: rule.Path, RunningKnown: processErr == nil}
		row.Running = appRunning(rule, processes)
		if m := meterFor(s.apps, rule); m != nil {
			if m.path != "" {
				row.Path = m.path
			}
			row.DownBps, row.UpBps = m.rateDown, m.rateUp
			row.DownBytes, row.UpBytes = m.down, m.up
			row.Warn, row.Capped = m.warn, m.capped
			row.Down = append([]float64(nil), m.graph...)
			row.Up = append([]float64(nil), m.graphUp...)
		}
		if running {
			row.Via = s.openAdapterNames(row.Exe)
		}
		out.Apps = append(out.Apps, row)
	}
	if !running {
		out.DownBps, out.UpBps = 0, 0
		for i := range out.Adapters {
			out.Adapters[i].DownBps, out.Adapters[i].UpBps = 0, 0
		}
		for i := range out.Apps {
			out.Apps[i].DownBps, out.Apps[i].UpBps = 0, 0
		}
	}
	s.mu.Unlock()
	// Enrich the copied status after releasing the data-plane lock.
	byName := s.adapterCache.get()
	for i := range out.Adapters {
		card := &out.Adapters[i]
		if card.Routable {
			if ifc, ok := byName[card.Name]; ok {
				receive, transmit := nics.LinkSpeeds(ifc.Index)
				card.LinkDownMbps, card.LinkUpMbps = float64(receive)/1e6, float64(transmit)/1e6
			}
		} else {
			if ifc, ok := byName[card.Name]; ok {
				card.IP, card.IP6 = s.adapterCache.ips(ifc)
			}
		}
	}
	return out
}

func (s *Service) PutRules(list []rules.Rule) error {
	if s.db != nil {
		if err := s.db.SaveRules(list); err != nil {
			return err
		}
	}
	s.rules.Store(&list)
	return nil
}

func (s *Service) PutSettings(st store.Settings) error {
	for _, limits := range []map[string]float64{st.AdapterDown, st.AdapterUp} {
		for _, v := range limits {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > engine.LimitMaxMbps {
				return fmt.Errorf("adapter limits must be between 0 and %g Mbit/s", engine.LimitMaxMbps)
			}
		}
	}
	st.AutoUpdate = false

	if st.Theme != "light" {
		st.Theme = "dark"
	}
	if s.db != nil {
		if err := s.db.SaveSettings(st); err != nil {
			return err
		}
	}
	s.settings.Store(&st)
	return nil
}

func (s *Service) Start() error {
	s.mu.Lock()
	sw := s.sw
	s.mu.Unlock()
	if sw == nil {
		return errNoEngine
	}
	err := sw.On()
	if err != nil {
		s.SetError(err.Error())
		return err
	}
	s.SetError("")
	s.paused.Store(false)
	return nil
}

func (s *Service) Stop() {
	s.paused.Store(true)
	s.mu.Lock()
	sw := s.sw
	s.recording = false
	s.openAdapters = nil
	s.trafficRates, s.trafficPending = nil, nil
	for _, m := range s.apps {
		m.rateDown, m.rateUp = 0, 0
		m.lastDown, m.lastUp = m.down, m.up
	}
	s.mu.Unlock()
	if sw != nil {
		sw.Off()
	}
	s.Flush()
	// Tray commands may subsequently resume the same switch directly.
	s.paused.Store(false)
}

// Quit turns routing off and ends this process. Stop only pauses routing.
func (s *Service) Quit() {
	s.Shutdown()
	os.Exit(0)
}

// Shutdown releases the kept device as well as the current capture generation.
func (s *Service) Shutdown() {
	s.Stop()
	s.mu.Lock()
	sw := s.sw
	s.mu.Unlock()
	if closer, ok := sw.(interface{ Close() }); ok {
		closer.Close()
	}
	if s.curlLog != nil {
		s.curlLog.close()
	}
}

func (s *Service) Flush() {
	if s.db == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for exe, m := range s.apps {
		down, up := m.down-m.flushedDown, m.up-m.flushedUp
		if down == 0 && up == 0 {
			continue
		}
		if err := s.db.AddSample(now, exe, m.path, m.via, down, up); err == nil {
			m.flushedDown, m.flushedUp = m.down, m.up
		}
	}
	for name, u := range s.uses {
		_ = s.db.SaveAdapterBytes(name, u.dayKey, u.todayDown, u.todayUp)
		_ = s.db.SaveAdapterBytes(name, u.monthKey, u.monthDown, u.monthUp)
	}
}

func (s *Service) Usage(from time.Time) ([]store.UsageRow, error) {
	if s.db == nil {
		return nil, nil
	}
	return s.db.Usage(from)
}

func (s *Service) UsageCSV(from time.Time) (string, error) {
	rows, err := s.Usage(from)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"exe", "path", "down_bytes", "up_bytes", "total_bytes"})
	for _, row := range rows {
		_ = w.Write([]string{row.Exe, row.Path, itoa(row.Down), itoa(row.Up), itoa(row.Down + row.Up)})
	}
	w.Flush()
	return b.String(), w.Error()
}

type linkReading struct {
	index    int
	in, out  uint64
	ok       bool
	hardware string
}

func (s *Service) sampleLink(name string, reading linkReading) (float64, float64) {
	in, out, ok := reading.in, reading.out, reading.ok
	prev := s.linkPrev[name]
	next := linkSample{in: in, out: out, primed: ok, index: reading.index, hardware: reading.hardware}
	if ok && prev.primed && prev.index == reading.index && prev.hardware == reading.hardware && in >= prev.in && out >= prev.out {
		sec := s.sampleSeconds
		if sec <= 0 {
			sec = 1
		}
		next.rateIn = float64(in-prev.in) / sec
		next.rateOut = float64(out-prev.out) / sec
	}
	s.linkPrev[name] = next
	return next.rateIn, next.rateOut
}

func vpnAdapter(name, reason string) bool {
	blob := strings.ToLower(name + " " + reason)
	for _, word := range []string{"vpn", "tailscale", "wireguard", "tunnel", "zerotier", "openvpn"} {
		if strings.Contains(blob, word) {
			return true
		}
	}
	return false
}

func hideAdapter(name, reason string) bool {
	blob := strings.ToLower(name + " " + reason)
	for _, word := range []string{"wsl", "docker", "vethernet", "loopback", "bluetooth", "npcap", "teredo", "isatap", "pseudo", "multiroutenetwork", "net-merge", "wintun"} {
		if strings.Contains(blob, word) {
			return true
		}
	}
	return false
}

func interfaceIPs(ifc net.Interface) (string, string) {
	addrs, err := ifc.Addrs()
	if err != nil {
		return "", ""
	}
	var v4, v6 string
	for _, addr := range addrs {
		ip, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip.IP.To4() != nil && v4 == "" && !ip.IP.IsLinkLocalUnicast() {
			v4 = ip.IP.String()
		} else if ip.IP.To4() == nil && v6 == "" && ip.IP.IsGlobalUnicast() {
			v6 = ip.IP.String()
		}
	}
	return v4, v6
}

func onlineState(state string) string {
	if state == "up" || state == "checking" {
		return "online"
	}
	if state == "" || state == "-" {
		return "online"
	}
	return state
}

func (s *Service) addAdapterBytes(name string, down, up int64) {
	if name == "" {
		return
	}
	now := time.Now()
	day, month := now.Format("2006-01-02"), now.Format("2006-01")
	u := s.uses[name]
	if u == nil {
		u = &adapterUse{}
		s.uses[name] = u
	}
	if !u.loaded {
		u.loaded = true
		u.dayKey, u.monthKey = day, month
		if s.db != nil {
			u.todayDown, u.todayUp, _ = s.db.AdapterBytes(name, day)
			u.monthDown, u.monthUp, _ = s.db.AdapterBytes(name, month)
		}
	}
	if u.dayKey != day {
		u.dayKey, u.todayDown, u.todayUp = day, 0, 0
	}
	if u.monthKey != month {
		u.monthKey, u.monthDown, u.monthUp = month, 0, 0
	}
	u.todayDown += down
	u.todayUp += up
	u.monthDown += down
	u.monthUp += up
}

func (s *Service) fillUse(card *AdapterCard) {
	s.addAdapterBytes(card.Name, 0, 0)
	u := s.uses[card.Name]
	if u == nil {
		return
	}
	card.TodayDown, card.TodayUp = u.todayDown, u.todayUp
	card.MonthDown, card.MonthUp = u.monthDown, u.monthUp
}

func push(s []float64, v float64) {
	copy(s, s[1:])
	s[len(s)-1] = v
}

func periodStart(period string, now time.Time) time.Time {
	local := now.In(time.Local)
	if period == "day" {
		return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
	}
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.Local)
}

func meterFor(apps map[string]*appMeter, rule rules.Rule) *appMeter {
	if rule.ExactPath && rule.Path != "" {
		for _, m := range apps {
			if strings.EqualFold(m.path, rule.Path) {
				return m
			}
		}
		return nil
	}
	for exe, m := range apps {
		if strings.EqualFold(exe, rule.Exe) {
			return m
		}
	}
	return nil
}

func matchRule(list []rules.Rule, h engine.FlowHint) (rules.Rule, bool) {
	res := rules.Choose(list, rules.Proc{Name: h.Name, Path: h.Path, Known: h.Known}, nil, nil, time.Now())
	_ = res
	for _, r := range list {
		if r.ExactPath && h.Path != "" && strings.EqualFold(r.Path, h.Path) {
			return r, true
		}
	}
	for _, r := range list {
		if !r.ExactPath && strings.EqualFold(r.Exe, h.Name) {
			return r, true
		}
	}
	return rules.Rule{}, false
}

func dnsHost(r rules.Rule) string {
	switch strings.ToLower(r.DNS) {
	case "cloudflare":
		return "1.1.1.1"
	case "google":
		return "8.8.8.8"
	case "quad9":
		return "9.9.9.9"
	case "custom":
		return r.CustomDNS
	default:
		return ""
	}
}

// trackAdapter records successful TCP/UDP socket lifetimes, including idle sockets.
func (s *Service) trackAdapter(exe, adapter string) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.openAdapters == nil {
		s.openAdapters = map[string]map[string]int{}
	}
	key := strings.ToLower(exe)
	counts := s.openAdapters[key]
	if counts == nil {
		counts = map[string]int{}
		s.openAdapters[key] = counts
	}
	counts[adapter]++
	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			counts[adapter]--
			if counts[adapter] <= 0 {
				delete(counts, adapter)
			}
		})
	}
}

// Called with the service mutex held. Sorting prevents changes caused by map order.
func (s *Service) openAdapterNames(exe string) string {
	names := []string{}
	for name, count := range s.openAdapters[strings.ToLower(exe)] {
		if count > 0 {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		ri, rj := s.trafficRates[strings.ToLower(exe)][names[i]], s.trafficRates[strings.ToLower(exe)][names[j]]
		if ri != rj {
			return ri > rj
		}
		a, b := strings.ToLower(names[i]), strings.ToLower(names[j])
		if a == b {
			return names[i] < names[j]
		}
		return a < b
	})
	return strings.Join(names, ", ")
}

func appRunning(rule rules.Rule, processes []procid.Proc) bool {
	for _, p := range processes {
		if rule.ExactPath {
			if p.Path != "" && strings.EqualFold(p.Path, rule.Path) {
				return true
			}
		} else if strings.EqualFold(p.Name, rule.Exe) {
			return true
		}
	}
	return false
}

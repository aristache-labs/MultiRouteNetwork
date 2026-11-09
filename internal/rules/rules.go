package rules

import (
	"strings"
	"time"
)

// Mode is how a rule picks adapters.
type Mode string

const (
	ModeAll     Mode = "all"
	ModeOne     Mode = "one"
	ModeSeveral Mode = "several"
)

// Fallback decides what happens when the chosen adapters are down.
type Fallback string

const (
	FallbackWait    Fallback = "wait"
	FallbackOthers  Fallback = "others"
	FallbackDefault Fallback = "default"
	FallbackChoose  Fallback = "choose"
)

// Rule is one per-app routing rule. Empty numeric limits mean unlimited.
type Rule struct {
	ID               string             `json:"id"`
	Exe              string             `json:"exe"`
	Path             string             `json:"path,omitempty"`
	ExactPath        bool               `json:"exactPath"`
	Mode             Mode               `json:"mode"`
	Adapters         []string           `json:"adapters,omitempty"`
	Preferred        string             `json:"preferred,omitempty"`
	Fallback         Fallback           `json:"fallback"`
	FallbackAdapters []string           `json:"fallbackAdapters,omitempty"`
	LastResort       bool               `json:"lastResort,omitempty"`
	LowestLatency    bool               `json:"lowestLatency"`
	DownMbps         float64            `json:"downMbps,omitempty"`
	UpMbps           float64            `json:"upMbps,omitempty"`
	AdapterDown      map[string]float64 `json:"adapterDown,omitempty"`
	AdapterUp        map[string]float64 `json:"adapterUp,omitempty"`
	CapGB            float64            `json:"capGB,omitempty"`
	CapPeriod        string             `json:"capPeriod,omitempty"` // day or month
	DNS              string             `json:"dns,omitempty"`       // windows, cloudflare, google, quad9, custom
	CustomDNS        string             `json:"customDNS,omitempty"`
}

// Proc is a resolved process. Known is false when the owner could not be identified.
type Proc struct {
	Name  string
	Path  string
	Known bool
}

// Adapter is a live candidate.
type Adapter struct {
	Name    string
	Default bool
	Up      bool
	Latency time.Duration
}

// Result is what the engine should do for one new flow.
type Result struct {
	Balance bool
	Block   bool
	Pin     string
	Only    []string
	DNS     string
}

// Hold remembers the last lowest-latency choice so it does not flap.
type Hold struct {
	Key  string
	Name string
	Till time.Time
}

const holdFor = 8 * time.Second

// Choose matches proc to a rule and picks the adapter set.
// An unknown process, or a process with no rule, balances across every candidate.
func Choose(list []Rule, proc Proc, ads []Adapter, hold *Hold, now time.Time) Result {
	if !proc.Known {
		return Result{Balance: true}
	}
	rule, ok := match(list, proc)
	if !ok {
		return Result{Balance: true}
	}
	allowed := selected(rule)
	pool := usable(ads, allowed, len(allowed) > 0)
	if len(pool) == 0 {
		switch rule.Fallback {
		case FallbackDefault:
			for _, a := range ads {
				if a.Up && a.Default {
					return Result{Pin: a.Name, DNS: dnsOf(rule)}
				}
			}
			return Result{Block: true, DNS: dnsOf(rule)}
		case FallbackChoose:
			for _, name := range rule.FallbackAdapters {
				for _, a := range ads {
					if a.Up && strings.EqualFold(a.Name, name) {
						return Result{Pin: a.Name, DNS: dnsOf(rule)}
					}
				}
			}
			if !rule.LastResort {
				return Result{Block: true, DNS: dnsOf(rule)}
			}
			pool = usable(ads, nil, false)
		}
	}
	if len(pool) == 0 && rule.Fallback == FallbackOthers {
		pool = usable(ads, nil, false)
	}
	if len(pool) == 0 {
		if rule.Fallback == FallbackWait || rule.Fallback == "" {
			return Result{Block: true, DNS: dnsOf(rule)}
		}
		return Result{Balance: true, DNS: dnsOf(rule)}
	}
	names := make([]string, len(pool))
	for i, a := range pool {
		names[i] = a.Name
	}
	if rule.LowestLatency {
		name := stick(hold, proc, pool, now)
		return Result{Pin: name, DNS: dnsOf(rule)}
	}
	if rule.Mode == ModeOne {
		want := rule.Preferred
		if rule.Mode == ModeOne && len(rule.Adapters) > 0 {
			want = rule.Adapters[0]
		}
		if contains(names, want) {
			return Result{Pin: want, DNS: dnsOf(rule)}
		}
	}
	if rule.Mode == ModeSeveral || (rule.Mode == ModeAll && len(allowed) > 0) {
		return Result{Only: names, DNS: dnsOf(rule)}
	}
	return Result{Only: names, DNS: dnsOf(rule)}
}

func match(list []Rule, proc Proc) (Rule, bool) {
	var byName *Rule
	for i := range list {
		r := &list[i]
		if r.ExactPath {
			if proc.Path != "" && strings.EqualFold(r.Path, proc.Path) {
				return *r, true
			}
			continue
		}
		if byName == nil && strings.EqualFold(r.Exe, proc.Name) {
			byName = r
		}
	}
	if byName != nil {
		return *byName, true
	}
	return Rule{}, false
}

func selected(r Rule) []string {
	switch r.Mode {
	case ModeOne, ModeSeveral:
		return r.Adapters
	default:
		return nil
	}
}

func usable(ads []Adapter, allow []string, restrict bool) []Adapter {
	var out []Adapter
	for _, a := range ads {
		if !a.Up {
			continue
		}
		if restrict && len(allow) > 0 && !contains(allow, a.Name) {
			continue
		}
		out = append(out, a)
	}
	return out
}

func stick(hold *Hold, proc Proc, pool []Adapter, now time.Time) string {
	key := proc.Path
	if key == "" {
		key = proc.Name
	}
	if hold != nil && hold.Key == key && now.Before(hold.Till) && containsAdapter(pool, hold.Name) {
		return hold.Name
	}
	best := pool[0]
	for _, a := range pool[1:] {
		if a.Latency > 0 && (best.Latency == 0 || a.Latency < best.Latency) {
			best = a
		}
	}
	if hold != nil {
		hold.Key, hold.Name, hold.Till = key, best.Name, now.Add(holdFor)
	}
	return best.Name
}

func contains(list []string, name string) bool {
	for _, s := range list {
		if strings.EqualFold(s, name) {
			return true
		}
	}
	return false
}

func containsAdapter(list []Adapter, name string) bool {
	for _, a := range list {
		if strings.EqualFold(a.Name, name) {
			return true
		}
	}
	return false
}

func dnsOf(r Rule) string {
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

// CapKey identifies one app for a daily or monthly counter.
func CapKey(exe, period string, now time.Time) string {
	if period == "day" {
		return strings.ToLower(exe) + "|" + now.Format("2006-01-02")
	}
	return strings.ToLower(exe) + "|" + now.Format("2006-01")
}

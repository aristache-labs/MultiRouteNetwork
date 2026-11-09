package control

import (
	"net"
	"sync"
	"time"
)

const adapterIdentityTTL = 3 * time.Second

// Published interface maps are immutable. This lock is separate from transfer
// accounting; a slow Windows enumeration never holds Service.mu.
type adapterIdentityCache struct {
	mu        sync.Mutex
	list      func() ([]net.Interface, error)
	now       func() time.Time
	expires   time.Time
	byName    map[string]net.Interface
	addresses map[string]cachedAddresses
}
type cachedAddresses struct {
	index   int
	expires time.Time
	v4, v6  string
}

func newAdapterIdentityCache(list func() ([]net.Interface, error)) *adapterIdentityCache {
	return &adapterIdentityCache{list: list, now: time.Now, addresses: map[string]cachedAddresses{}}
}
func (c *adapterIdentityCache) get() map[string]net.Interface {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byName != nil && c.now().Before(c.expires) {
		return c.byName
	}
	interfaces, err := c.list()
	next := map[string]net.Interface{}
	if err != nil {
		// Avoid associating counters with stale indices after an enumeration failure.
		c.byName = next
		c.addresses = map[string]cachedAddresses{}
		c.expires = c.now().Add(time.Second)
		return next
	}
	for _, ifc := range interfaces {
		next[ifc.Name] = ifc
	}
	c.byName = next
	c.expires = c.now().Add(adapterIdentityTTL)
	c.addresses = map[string]cachedAddresses{}
	return next
}
func (c *adapterIdentityCache) ips(ifc net.Interface) (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if a, ok := c.addresses[ifc.Name]; ok && a.index == ifc.Index && now.Before(a.expires) {
		return a.v4, a.v6
	}
	// Only visible non-routing cards need this lookup; never query every virtual
	// interface's addresses on each refresh.
	v4, v6 := interfaceIPs(ifc)
	c.addresses[ifc.Name] = cachedAddresses{index: ifc.Index, expires: c.now().Add(adapterIdentityTTL), v4: v4, v6: v6}
	return v4, v6
}

package control

import (
	"math"
	"strconv"
	"time"
)

type bucket struct {
	mbps   float64
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newBucket(mbps float64) *bucket {
	rate := mbps * 1_000_000 / 8
	burst := math.Max(16<<10, rate*0.05)
	return &bucket{mbps: mbps, rate: rate, burst: burst, tokens: burst, last: time.Now()}
}

func (b *bucket) take(n int) time.Duration {
	now := time.Now()
	b.tokens = math.Min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.rate)
	b.last = now
	var wait time.Duration
	if need := float64(n); need > b.tokens {
		seconds := (need - b.tokens) / b.rate
		b.tokens = 0
		b.last = now.Add(time.Duration(seconds * float64(time.Second)))
		wait = time.Duration(seconds * float64(time.Second))
	} else {
		b.tokens -= need
	}
	return wait
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

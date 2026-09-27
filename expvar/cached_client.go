package expvar

import (
	"context"
	"sync"
	"time"

	"github.com/grum261/xray-exporter/internal/singleflight"
)

// Snapshotter is implemented by anything that can fetch a Snapshot.
// Defined here so consumers can depend on the interface, not *Client.
type Snapshotter interface {
	Fetch(ctx context.Context) (Snapshot, error)
}

// CachedClient wraps a Snapshotter and coalesces concurrent Fetch calls
// into a single upstream request via singleflight. A prometheus Registry
// invokes its registered Collectors concurrently during Gather; without
// this wrapper each domain-specific Collector would trigger its own HTTP
// scrape per cycle.
type CachedClient struct {
	upstream Snapshotter
	group    singleflight.Group[Snapshot]

	mu           sync.RWMutex
	lastDuration time.Duration
	lastErr      error
	lastAt       time.Time
}

// NewCachedClient wraps upstream. Returns an error if upstream is nil.
func NewCachedClient(upstream Snapshotter) (*CachedClient, error) {
	if upstream == nil {
		return nil, ErrUpstreamRequired
	}
	return &CachedClient{upstream: upstream}, nil
}

// Fetch invokes the upstream's Fetch, coalescing concurrent callers.
// Only the first caller in a coalesced batch performs the HTTP request;
// others receive the same Snapshot/error.
func (c *CachedClient) Fetch(ctx context.Context) (Snapshot, error) {
	snap, _, err := c.group.Do("fetch", func() (Snapshot, error) {
		start := time.Now()
		s, err := c.upstream.Fetch(ctx)
		c.mu.Lock()
		c.lastDuration = time.Since(start)
		c.lastErr = err
		c.lastAt = time.Now()
		c.mu.Unlock()
		return s, err
	})
	return snap, err
}

// LastFetch returns the duration of the most recent upstream fetch, whether
// any fetch has occurred yet, and its error. Used by ScrapeCollector to emit
// health metrics without triggering an extra HTTP request.
func (c *CachedClient) LastFetch() (duration time.Duration, ok bool, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastDuration, !c.lastAt.IsZero(), c.lastErr
}

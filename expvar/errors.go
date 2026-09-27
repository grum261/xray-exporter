package expvar

import "errors"

// ErrScrapeFailed indicates the scrape failed for any reason — network,
// status code, or malformed body. Returned scrape errors wrap it; detect
// with errors.Is.
var ErrScrapeFailed = errors.New("xray scrape failed")

// ErrEndpointRequired is returned by NewClient when no endpoint is configured.
var ErrEndpointRequired = errors.New("endpoint is required")

// ErrUpstreamRequired is returned by NewCachedClient when upstream is nil.
var ErrUpstreamRequired = errors.New("upstream is required")

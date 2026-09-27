package expvar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxResponseBytes caps the body read to protect against pathological responses.
// xray /debug/vars is typically <50 KB even with many tags.
const maxResponseBytes int64 = 5 * 1024 * 1024

// Client fetches and parses /debug/vars from xray.
type Client struct {
	endpoint string
	http     *http.Client
}

// NewClient creates a Client. Options override defaults; without any
// options the client targets defaultEndpoint with defaultTimeout.
func NewClient(opts ...Option) (*Client, error) {
	cfg := defaultClientConfig()
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}
	if cfg.endpoint == "" {
		return nil, ErrEndpointRequired
	}
	return &Client{
		endpoint: cfg.endpoint,
		http:     httpClient,
	}, nil
}

// Fetch retrieves the current /debug/vars snapshot and parses it.
// Errors are wrapped with ErrScrapeFailed for the caller to detect.
func (c *Client) Fetch(ctx context.Context) (Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return Snapshot{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: %w", ErrScrapeFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return Snapshot{}, fmt.Errorf("%w: unexpected status %d", ErrScrapeFailed, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: read body: %w", ErrScrapeFailed, err)
	}

	snap, err := Parse(body)
	if err != nil {
		return Snapshot{}, err
	}

	snap.ScrapedAt = time.Now()

	return snap, nil
}

// Parse decodes a raw /debug/vars JSON body into a Snapshot.
// Exported for testing.
func Parse(body []byte) (Snapshot, error) {
	var snap Snapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		return Snapshot{}, fmt.Errorf("%w: parse json: %w", ErrScrapeFailed, err)
	}
	return snap, nil
}

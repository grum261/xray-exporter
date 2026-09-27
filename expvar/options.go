package expvar

import (
	"net/http"
	"time"

	"github.com/grum261/xray-exporter/option"
)

const (
	defaultEndpoint = "http://127.0.0.1:11111/debug/vars"
	defaultTimeout  = 5 * time.Second
)

// clientConfig is the resolved Client configuration assembled from Options.
type clientConfig struct {
	endpoint   string
	timeout    time.Duration
	httpClient *http.Client
}

func defaultClientConfig() *clientConfig {
	return &clientConfig{
		endpoint: defaultEndpoint,
		timeout:  defaultTimeout,
	}
}

// Option configures a Client. Option is parameterized over the unexported
// clientConfig, so only the WithXxx constructors in this package can
// produce values that satisfy it.
type Option = option.Option[clientConfig]

// WithEndpoint overrides the xray /debug/vars URL.
func WithEndpoint(endpoint string) Option {
	return option.New(func(c *clientConfig) {
		if endpoint != "" {
			c.endpoint = endpoint
		}
	})
}

// WithTimeout overrides the per-request timeout. Applied to the default
// http.Client if WithHTTPClient was not provided.
func WithTimeout(d time.Duration) Option {
	return option.New(func(c *clientConfig) {
		if d > 0 {
			c.timeout = d
		}
	})
}

// WithHTTPClient overrides the http.Client used for scrapes. Takes
// precedence over WithTimeout — set timeout on the supplied client.
func WithHTTPClient(h *http.Client) Option {
	return option.New(func(c *clientConfig) {
		if h != nil {
			c.httpClient = h
		}
	})
}

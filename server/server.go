// Package server wires the xray-exporter HTTP endpoint and registers
// prometheus.Collectors handed to it. The Server is constructed with
// New(opts...) following the grpc-go option pattern.
//
// The server is intentionally thin: it does not know about expvar
// clients or specific collectors. The caller (typically main) acts as
// the composition root, building collectors and passing them in via
// WithCollectors.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/grum261/xray-exporter/logger"
)

// Server is the xray-exporter HTTP server. Build it with New(opts...) and
// run it via Run(ctx).
type Server struct {
	cfg    *config
	srv    *http.Server
	logger logger.Logger
}

// New constructs a Server and registers the supplied collectors with a
// fresh prometheus.Registry. Returns an error if no collector was
// supplied — the exporter would expose nothing useful in that state.
func New(opts ...Option) (*Server, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.Apply(cfg)
	}

	if len(cfg.collectors) == 0 {
		return nil, ErrNoCollectors
	}

	registry := prometheus.NewRegistry()
	for i := range cfg.collectors {
		if err := registry.Register(cfg.collectors[i]); err != nil {
			return nil, fmt.Errorf("register collector %d: %w", i, err)
		}
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		Registry:      registry,
		ErrorHandling: promhttp.ContinueOnError,
	}))
	mux.HandleFunc("/-/healthy", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok\n")
	})
	mux.HandleFunc("/", indexHandler(cfg.metricsPath))

	httpSrv := &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: cfg.readHeaderTimeout,
	}

	cfg.logger.Info("registered collectors", "count", len(cfg.collectors))

	return &Server{cfg: cfg, srv: httpSrv, logger: cfg.logger}, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled or the
// server fails. On cancellation it performs a graceful shutdown bounded
// by the configured shutdown timeout.
func (s *Server) Run(ctx context.Context) error {
	serverErr := make(chan error, 1)
	go func() {
		s.logger.Info("listening", "addr", s.cfg.listenAddr, "metrics_path", s.cfg.metricsPath)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("shutting down")
	case err := <-serverErr:
		return err
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), s.cfg.shutdownTimeout)
	defer cancel()

	if err := s.srv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	return nil
}

// indexHandler serves a tiny landing page pointing at the metrics endpoint.
func indexHandler(metricsPath string) http.HandlerFunc {
	body := fmt.Sprintf(`<html>
<head><title>xray-exporter</title></head>
<body>
<h1>xray-exporter</h1>
<p><a href="%s">Metrics</a></p>
<p><a href="/-/healthy">Health</a></p>
</body>
</html>`, metricsPath)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, body)
	}
}

package server

import "errors"

// ErrNoCollectors is returned by New when no collector was supplied — the
// exporter would expose nothing useful in that state.
var ErrNoCollectors = errors.New("at least one collector is required (use WithCollectors)")

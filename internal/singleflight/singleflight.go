// Package singleflight is a generic, type-safe wrapper around
// golang.org/x/sync/singleflight. It coalesces concurrent calls keyed by a
// string so only one execution runs at a time; duplicate callers receive
// the same result, without the any-typed return and type assertion of the
// underlying package.
package singleflight

import "golang.org/x/sync/singleflight"

// Group coalesces concurrent calls returning T. The zero Group is ready to
// use and must not be copied after first use.
type Group[T any] struct {
	g singleflight.Group
}

// Result is the outcome delivered over the channel returned by DoChan.
type Result[T any] struct {
	Val    T
	Err    error
	Shared bool
}

// Do executes fn, making sure only one execution is in flight for key at a
// time. Duplicate callers block until the leader returns, then share its
// value, shared=true, and error. The returned value is fn's result, typed
// as T.
func (g *Group[T]) Do(key string, fn func() (T, error)) (v T, shared bool, err error) {
	raw, err, shared := g.g.Do(key, func() (any, error) {
		return fn()
	})
	v, _ = raw.(T)
	return v, shared, err
}

// DoChan is like Do but returns a channel that receives the result when fn
// completes. The returned channel is not closed.
func (g *Group[T]) DoChan(key string, fn func() (T, error)) <-chan Result[T] {
	out := make(chan Result[T], 1)
	ch := g.g.DoChan(key, func() (any, error) {
		return fn()
	})
	go func() {
		r := <-ch
		v, _ := r.Val.(T)
		out <- Result[T]{Val: v, Err: r.Err, Shared: r.Shared}
	}()
	return out
}

// Forget tells the Group to forget key, so a future call will execute fn
// again rather than waiting for an in-flight leader to finish.
func (g *Group[T]) Forget(key string) {
	g.g.Forget(key)
}

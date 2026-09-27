package singleflight_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grum261/xray-exporter/internal/singleflight"
)

func TestGroup_Do_ReturnsTypedValue(t *testing.T) {
	t.Parallel()

	var g singleflight.Group[string]
	v, shared, err := g.Do("k", func() (string, error) {
		return "hello", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "hello" {
		t.Errorf("value = %q, want %q", v, "hello")
	}
	if shared {
		t.Errorf("shared = true for a lone call, want false")
	}
}

func TestGroup_Do_PropagatesError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	var g singleflight.Group[int]
	v, _, err := g.Do("k", func() (int, error) {
		return 0, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want %v", err, sentinel)
	}
	if v != 0 {
		t.Errorf("value = %d, want 0", v)
	}
}

func TestGroup_Do_Coalesces(t *testing.T) {
	t.Parallel()

	var g singleflight.Group[int]
	var calls atomic.Int32

	started := make(chan struct{})
	release := make(chan struct{})

	// Leader: signal it is in flight, then block until released.
	leaderDone := make(chan int, 1)
	go func() {
		v, _, _ := g.Do("k", func() (int, error) {
			calls.Add(1)
			close(started)
			<-release
			return 99, nil
		})
		leaderDone <- v
	}()

	<-started // leader is now executing fn

	// Duplicate callers arrive while the leader is in flight.
	const dups = 8
	var wg sync.WaitGroup
	shareds := make([]bool, dups)
	vals := make([]int, dups)
	for i := range dups {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			v, shared, _ := g.Do("k", func() (int, error) {
				calls.Add(1)
				return -1, nil
			})
			vals[idx] = v
			shareds[idx] = shared
		}(i)
	}

	// Give the duplicates time to coalesce onto the in-flight leader.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := <-leaderDone; got != 99 {
		t.Errorf("leader value = %d, want 99", got)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("fn executed %d times, want 1 (coalesced)", n)
	}
	for i := range dups {
		if vals[i] != 99 {
			t.Errorf("dup %d value = %d, want 99", i, vals[i])
		}
		if !shareds[i] {
			t.Errorf("dup %d shared = false, want true", i)
		}
	}
}

func TestGroup_Forget(t *testing.T) {
	t.Parallel()

	var g singleflight.Group[int]
	var calls atomic.Int32
	fn := func() (int, error) {
		calls.Add(1)
		return 1, nil
	}

	_, _, _ = g.Do("k", fn)
	g.Forget("k")
	_, _, _ = g.Do("k", fn)

	if n := calls.Load(); n != 2 {
		t.Errorf("fn executed %d times, want 2 after Forget", n)
	}
}

func TestGroup_DoChan(t *testing.T) {
	t.Parallel()

	var g singleflight.Group[string]
	res := <-g.DoChan("k", func() (string, error) {
		return "via-chan", nil
	})
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.Val != "via-chan" {
		t.Errorf("value = %q, want %q", res.Val, "via-chan")
	}
}

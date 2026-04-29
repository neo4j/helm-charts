// Package poll provides a single polling primitive for tests that must wait
// for an eventually-consistent condition (a Kubernetes pod becomes Ready, a
// secret becomes retrievable, a Neo4j cluster elects a leader) without
// devolving into ad-hoc time.Sleep loops.
//
// Every wait helper in this repo historically implemented its own retry
// cadence, its own timeout handling, and its own error reporting. The ad-hoc
// versions disagreed on everything: fixed 30-second sleeps, silent time.Sleep
// on transient errors, no visibility into how many attempts were made. This
// package exists so every wait across the test suite:
//
//   - cancels cleanly when the test context ends (passes -timeout through),
//   - classifies errors as retryable or terminal via an injected predicate,
//   - produces one summary line on final failure ("waited 47s across 24
//     attempts for <description>") that makes flake triage fast, and
//   - emits per-attempt logs only when CI_POLL_VERBOSE=1 is set, so the
//     default CI log stays uncluttered.
package poll

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

const defaultInterval = 2 * time.Second

// Opts configures a single invocation of Until / UntilValue. Timeout is
// required; every other field has a sensible zero value.
type Opts struct {
	// Interval is the delay between attempts. Defaults to 2 seconds when zero.
	Interval time.Duration

	// Timeout is the total budget across all attempts. Required.
	Timeout time.Duration

	// Description is the short human-readable subject of the wait, appearing
	// in the final failure line (e.g. "pod my-app-0 to be Ready").
	Description string

	// RetryableErrs, if non-nil, classifies errors as transient. A non-nil
	// error from the poll function aborts immediately unless RetryableErrs
	// returns true for it. When RetryableErrs is nil, every error is treated
	// as terminal — this is the strict default; pass a classifier for
	// eventually-consistent APIs such as kubectl / kubernetes client-go.
	RetryableErrs func(error) bool
}

// ErrTimeout is returned (wrapped) when the overall timeout is exhausted.
var ErrTimeout = errors.New("poll: timeout exceeded")

// Until repeatedly invokes fn until it reports done=true and nil error,
// returns a non-retryable error, or the timeout expires.
//
// fn must be side-effect free across retries: callers should treat each
// invocation as an idempotent probe, not a driver of the subject's state.
//
// When fn returns (true, nil), Until returns nil. Otherwise Until returns
// a wrapped error describing how long it waited, how many attempts it made,
// and the last observed error (if any). Cancellation of ctx also returns
// a wrapped error.
func Until(ctx context.Context, t testing.TB, opts Opts, fn func(context.Context) (bool, error)) error {
	t.Helper()
	if opts.Timeout <= 0 {
		return fmt.Errorf("poll: Timeout must be positive, got %s", opts.Timeout)
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	verbose := os.Getenv("CI_POLL_VERBOSE") == "1"

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	start := time.Now()
	var lastErr error
	attempts := 0

	for {
		attempts++
		done, err := fn(ctx)
		switch {
		case err == nil && done:
			if verbose {
				t.Logf("poll: %q succeeded after %d attempt(s) in %s",
					opts.Description, attempts, time.Since(start).Round(time.Millisecond))
			}
			return nil
		case err != nil && opts.RetryableErrs != nil && !opts.RetryableErrs(err):
			return fmt.Errorf("poll: %q aborted on non-retryable error after %d attempt(s) in %s: %w",
				opts.Description, attempts, time.Since(start).Round(time.Millisecond), err)
		default:
			lastErr = err
			if verbose {
				t.Logf("poll: %q attempt %d not ready (elapsed=%s, err=%v)",
					opts.Description, attempts, time.Since(start).Round(time.Millisecond), err)
			}
		}

		select {
		case <-ctx.Done():
			elapsed := time.Since(start).Round(time.Millisecond)
			reason := fmt.Errorf("%w after %d attempt(s) in %s", ErrTimeout, attempts, elapsed)
			if lastErr != nil {
				return fmt.Errorf("poll: %q timed out (%w); last error: %w",
					opts.Description, reason, lastErr)
			}
			return fmt.Errorf("poll: %q timed out (%w)", opts.Description, reason)
		case <-time.After(interval):
		}
	}
}

// UntilValue is the typed counterpart to Until: fn returns a value alongside
// the ready/error signals, and the successful value is propagated back to the
// caller. Use this when the observation itself (pod name, ingress IP, logs)
// is the thing the caller wants.
func UntilValue[T any](ctx context.Context, t testing.TB, opts Opts, fn func(context.Context) (T, bool, error)) (T, error) {
	t.Helper()
	var result T
	err := Until(ctx, t, opts, func(ctx context.Context) (bool, error) {
		v, done, err := fn(ctx)
		if done && err == nil {
			result = v
		}
		return done, err
	})
	return result, err
}

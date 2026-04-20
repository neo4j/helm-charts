package poll

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestUntilReturnsQuicklyWhenConditionAlreadyTrue(t *testing.T) {
	t.Parallel()
	start := time.Now()
	err := Until(context.Background(), t, Opts{Interval: 50 * time.Millisecond, Timeout: 5 * time.Second, Description: "always-true"},
		func(context.Context) (bool, error) { return true, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("Until should have returned immediately, took %s", elapsed)
	}
}

func TestUntilPollsUntilReady(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	err := Until(context.Background(), t, Opts{Interval: 10 * time.Millisecond, Timeout: 5 * time.Second, Description: "ready-after-3"},
		func(context.Context) (bool, error) {
			return attempts.Add(1) >= 3, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestUntilSurfacesNonRetryableErrorsImmediately(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("boom")
	var attempts atomic.Int32
	err := Until(context.Background(), t, Opts{
		Interval:      10 * time.Millisecond,
		Timeout:       1 * time.Second,
		Description:   "strict-mode",
		RetryableErrs: func(err error) bool { return errors.Is(err, context.Canceled) },
	}, func(context.Context) (bool, error) {
		attempts.Add(1)
		return false, sentinel
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel, got: %v", err)
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("expected immediate abort (1 attempt), got %d", got)
	}
}

func TestUntilTreatsErrorsAsRetryableByDefaultWhenClassifierRetries(t *testing.T) {
	t.Parallel()
	retryableErr := errors.New("transient")
	var attempts atomic.Int32
	err := Until(context.Background(), t, Opts{
		Interval:      10 * time.Millisecond,
		Timeout:       5 * time.Second,
		Description:   "flaky-then-ready",
		RetryableErrs: func(error) bool { return true },
	}, func(context.Context) (bool, error) {
		n := attempts.Add(1)
		if n < 3 {
			return false, retryableErr
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestUntilReportsTimeoutWithAttemptCountAndLastError(t *testing.T) {
	t.Parallel()
	lastErr := errors.New("still pending")
	err := Until(context.Background(), t, Opts{
		Interval:      20 * time.Millisecond,
		Timeout:       80 * time.Millisecond,
		Description:   "pod xyz to be Ready",
		RetryableErrs: func(error) bool { return true },
	}, func(context.Context) (bool, error) {
		return false, lastErr
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected wrapped ErrTimeout, got %v", err)
	}
	if !errors.Is(err, lastErr) {
		t.Fatalf("expected wrapped last error, got %v", err)
	}
	if !strings.Contains(err.Error(), "pod xyz to be Ready") {
		t.Fatalf("expected description in error, got %v", err)
	}
}

func TestUntilRespectsContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	err := Until(ctx, t, Opts{
		Interval:      10 * time.Millisecond,
		Timeout:       5 * time.Second,
		Description:   "never-ready",
		RetryableErrs: func(error) bool { return true },
	}, func(context.Context) (bool, error) {
		return false, nil
	})
	if err == nil {
		t.Fatal("expected error after cancellation")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected wrapped ErrTimeout, got %v", err)
	}
}

func TestUntilValuePropagatesResult(t *testing.T) {
	t.Parallel()
	got, err := UntilValue(context.Background(), t, Opts{Interval: 10 * time.Millisecond, Timeout: 2 * time.Second, Description: "42"},
		func(context.Context) (int, bool, error) {
			return 42, true, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestUntilRejectsNonPositiveTimeout(t *testing.T) {
	t.Parallel()
	err := Until(context.Background(), t, Opts{Timeout: 0, Description: "bad"},
		func(context.Context) (bool, error) { return true, nil })
	if err == nil {
		t.Fatal("expected error for zero timeout")
	}
	if !strings.Contains(err.Error(), "Timeout must be positive") {
		t.Fatalf("expected precondition error, got %v", err)
	}
}

package timeouts

import (
	"testing"
	"time"
)

func TestEnvDurationDefaultsWhenUnset(t *testing.T) {
	t.Parallel()
	got := envDuration("TEST_TIMEOUTS_UNSET_KEY_"+t.Name(), 42*time.Second)
	if got != 42*time.Second {
		t.Fatalf("expected 42s default, got %s", got)
	}
}

func TestEnvDurationParsesValidOverride(t *testing.T) {
	t.Setenv("TEST_TIMEOUTS_OVERRIDE", "7m")
	got := envDuration("TEST_TIMEOUTS_OVERRIDE", 42*time.Second)
	if got != 7*time.Minute {
		t.Fatalf("expected 7m override, got %s", got)
	}
}

func TestEnvDurationFallsBackOnParseFailure(t *testing.T) {
	t.Setenv("TEST_TIMEOUTS_BAD", "not-a-duration")
	got := envDuration("TEST_TIMEOUTS_BAD", 42*time.Second)
	if got != 42*time.Second {
		t.Fatalf("expected fallback to 42s on malformed override, got %s", got)
	}
}

func TestKubectlFlagFormatsSeconds(t *testing.T) {
	t.Parallel()
	if got := kubectlFlag(300 * time.Second); got != "--timeout=300s" {
		t.Fatalf("unexpected flag: %s", got)
	}
	if got := kubectlFlag(5 * time.Minute); got != "--timeout=300s" {
		t.Fatalf("unexpected flag for 5m: %s", got)
	}
}

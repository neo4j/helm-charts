// Package timeouts centralizes the hard-coded kubectl --timeout values that
// were previously scattered across integration_tests (180s, 120s, 300s strings
// repeated verbatim in six files). Timeouts live here so a slow-cluster day
// can be accommodated by flipping one env var rather than editing code.
package timeouts

import (
	"fmt"
	"os"
	"time"
)

// TestTimeouts is the suite-wide policy for how long kubectl / poll operations
// are allowed to run before surfacing a timeout. Each field can be overridden
// by an environment variable, which lets CI dial the envelope without code
// changes on known-slow days (e.g. when GKE regional capacity is tight).
var TestTimeouts = struct {
	// Rollout is used for kubectl rollout status --watch on StatefulSets
	// and Deployments. Historical value: 120-180s (varied by call site).
	Rollout time.Duration

	// PodReady is used for kubectl wait --for=condition=Ready/Initialized.
	// Historical value: 300s.
	PodReady time.Duration

	// Delete is used for kubectl delete --wait timeouts on StatefulSets,
	// Pods, and PVCs. Historical value: 120s.
	Delete time.Duration
}{
	Rollout:  envDuration("TEST_ROLLOUT_TIMEOUT", 180*time.Second),
	PodReady: envDuration("TEST_POD_READY_TIMEOUT", 300*time.Second),
	Delete:   envDuration("TEST_DELETE_TIMEOUT", 120*time.Second),
}

// KubectlRollout returns the --timeout=<X>s value for kubectl rollout status.
func KubectlRollout() string { return kubectlFlag(TestTimeouts.Rollout) }

// KubectlPodReady returns the --timeout=<X>s value for kubectl wait on pods.
func KubectlPodReady() string { return kubectlFlag(TestTimeouts.PodReady) }

// KubectlDelete returns the --timeout=<X>s value for kubectl delete --wait.
func KubectlDelete() string { return kubectlFlag(TestTimeouts.Delete) }

func kubectlFlag(d time.Duration) string {
	// kubectl accepts Go duration strings (e.g. "180s", "5m") — pass the
	// seconds form so callers can still eyeball them at a glance.
	return fmt.Sprintf("--timeout=%ds", int(d.Seconds()))
}

func envDuration(key string, def time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		// Fall back to the default rather than panicking — a typo in a CI
		// env var should not cascade into unrelated test failures.
		return def
	}
	return d
}

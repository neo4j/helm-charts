package integration_tests

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/testutil/testid"
)

var (
	// TestRunIdentifier is a run-scoped suffix shared by every test in one invocation
	// of `go test`. Use TestNamespace(t) for resource naming so parallel tests that
	// share a prefix cannot collide on namespaces, secrets, or buckets.
	TestRunIdentifier string
)

var Neo4jConfFile = fmt.Sprintf("neo4j/neo4j-%s.conf", model.Neo4jEdition)

func init() {
	// Format kept intentionally short: Kubernetes CronJob names are capped at
	// 52 chars and Helm release names at 53. TestNamespace() appends a
	// 7-char SHA suffix (`-` + 6 hex) on top of this run ID, so the combined
	// suffix must stay well under ~20 chars to leave room for descriptive
	// prefixes like "standalone-backup-local-incon-" (30 chars).
	dt := time.Now()
	dateTag := dt.Format("150405")
	randomSuffix := rand.Intn(1000)
	TestRunIdentifier = strings.ToLower(fmt.Sprintf("%s-%d", dateTag, randomSuffix))
}

// TestNamespace returns a deterministic, per-test identifier that embeds both
// the run identifier and a short hash of t.Name(). Two tests running in parallel
// in the same binary will always get different values, even if their release-name
// prefixes match — which eliminates the namespace-collision class of flakiness
// that previously required defensive "already exists" checks.
//
// Callers should continue to supply a human-readable prefix, e.g.:
//
//	releaseName := model.NewReleaseName("auth-wrong-key-" + TestNamespace(t))
func TestNamespace(t testing.TB) string {
	return testid.For(TestRunIdentifier, t)
}

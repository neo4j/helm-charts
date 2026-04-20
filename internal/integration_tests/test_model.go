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
	dt := time.Now()
	dateTag := dt.Format("15:04:05.00 Mon")
	randomSuffix := rand.Intn(1000)
	dateTag = fmt.Sprintf("%s-%d", dateTag, randomSuffix)
	dateTag = strings.ReplaceAll(dateTag, " ", "-")
	dateTag = strings.ReplaceAll(dateTag, ":", "-")
	dateTag = strings.ReplaceAll(dateTag, ".", "-")
	TestRunIdentifier = strings.ToLower(dateTag)
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

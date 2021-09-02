package integration_tests

import (
	"fmt"
	"k8s.io/utils/env"
	. "github.com/neo-technology/neo4j-helm-charts/internal/helpers"
	"strings"
	"testing"
	"time"
)

var (
	TestRunIdentifier string
)

var Neo4jEdition = strings.ToLower(env.GetString("NEO4J_EDITION", "enterprise"))
var Neo4jConfFile = fmt.Sprintf("neo4j-standalone/neo4j-%s.conf", Neo4jEdition)

func init() {
	dt := time.Now()
	dateTag := dt.Format("15:04:05 Mon")
	dateTag = strings.ReplaceAll(dateTag, " ", "-")
	dateTag = strings.ReplaceAll(dateTag, ":", "-")
	TestRunIdentifier = strings.ToLower(dateTag)
}

func cleanupTest(t *testing.T, cleanupWork Closeable) {

	t.Logf("Beginning cleanup of '%s'", t.Name())
	defer t.Logf("Finished cleanup of '%s'", t.Name())

	if cleanupWork != nil {
		err := cleanupWork()
		if err != nil {
			t.Errorf("Error during cleanup of test %s: %v", t.Name(), err)
			t.Fail()
		}
	}
}

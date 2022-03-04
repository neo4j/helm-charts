package integration_tests

import (
	"fmt"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/model"
	"strings"
	"testing"
	"time"
)

var (
	TestRunIdentifier string
)

var Neo4jConfFile = fmt.Sprintf("neo4j-standalone/neo4j-%s.conf", model.Neo4jEdition)

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

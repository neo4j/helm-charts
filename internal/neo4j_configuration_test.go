package internal

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func containsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Very quick test to check that no errors are thrown and a couple of values from the default neo4j conf show up
func TestPopulateFromFile(t *testing.T) {
	testCases := []string{
		"enterprise",
		"community",
	}

	edition, found := os.LookupEnv("NEO4J_EDITION")
	if found && !containsString(testCases, edition) {
		testCases = append(testCases, edition)
	}

	doTestCase := func(t *testing.T, edition string) {
		t.Parallel()
		conf, err := (&Neo4jConfiguration{}).PopulateFromFile(fmt.Sprintf("neo4j/neo4j-%s.conf", edition))
		if !assert.NoError(t, err) {
			return
		}

		value, found := conf.conf["dbms.windows_service_name"]
		assert.True(t, found)
		assert.Equal(t, "neo4j", value)

		_, jvmKeyFound := conf.conf[neo4jConfJvmAdditionalKey]
		assert.False(t, jvmKeyFound)

		assert.Contains(t, conf.jvmArgs, "-XX:+UnlockDiagnosticVMOptions")
		assert.Contains(t, conf.jvmArgs, "-XX:+DebugNonSafepoints")
		assert.Greater(t, len(conf.jvmArgs), 1)
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			doTestCase(t, testCase)
		})
	}
}

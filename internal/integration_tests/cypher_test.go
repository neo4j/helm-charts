package integration_tests

import (
	"errors"
	"fmt"
	"github.com/neo-technology/neo4j-helm-charts/internal/model"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/stretchr/testify/assert"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Auth stuff
const dbUri = "neo4j+ssc://localhost"
const user = "neo4j"
const dbName = "neo4j"

var authToUse = neo4j.BasicAuth(user, model.DefaultPassword, "")

// Track the total number of nodes that we've created
var createdNodes = map[model.ReleaseName]*int64{}

// empty param map (makes queries without params more readable)
var noParams = map[string]interface{}{}

func CheckNeo4jConfiguration(t *testing.T, releaseName model.ReleaseName, expectedConfiguration *model.Neo4jConfiguration) (err error) {

	var runtimeConfig []*neo4j.Record
	var expectedOverrides = map[string]string{
		"dbms.connector.https.enabled":  "true",
		"dbms.connector.bolt.tls_level": "REQUIRED",
		"dbms.directories.logs":         "/logs",
		"dbms.directories.metrics":      "/metrics",
		"dbms.directories.import":       "/import",
	}

	deadline := time.Now().Add(3 * time.Minute)
	for true {
		if !time.Now().Before(deadline) {
			msg := fmt.Sprintf("timed out fetching config:  %d", len(runtimeConfig))
			t.Error(msg)
			return errors.New(msg)
		}
		runtimeConfig, err = runQuery(t, releaseName, "CALL dbms.listConfig() YIELD name, value", nil)
		if err != nil {
			return err
		}
		if len(runtimeConfig) >= len(expectedConfiguration.Conf()) {
			break
		}
	}
	for key, value := range expectedOverrides {
		expectedConfiguration.Conf()[key] = value
	}

	for _, record := range runtimeConfig {
		nameUntyped, foundName := record.Get("name")
		valueUntyped, foundValue := record.Get("value")
		if !(foundName && foundValue) {
			return fmt.Errorf("record is missing expected name or value")
		}

		name := nameUntyped.(string)
		value := valueUntyped.(string)
		if expectedValue, found := expectedConfiguration.Conf()[name]; found {
			assert.Equal(t, strings.ToLower(expectedValue), strings.ToLower(value),
				"Expected runtime config for %s to match provided value", name)
		}
		if name == "dbms.jvm.additional" {
			assert.Equal(t, expectedConfiguration.JvmArgs(), strings.Split(value, "\n"))
		}
	}

	if err == nil {
		t.Log("Configuration check passed for:", releaseName.String())
	}
	return err
}

func CreateNode(t *testing.T, releaseName model.ReleaseName) error {
	_, err := runQuery(t, releaseName, "CREATE (n:Item { id: $id, name: $name }) RETURN n.id, n.name", map[string]interface{}{
		"id":   1,
		"name": "Item 1",
	})
	if _, found := createdNodes[releaseName]; !found {
		var initialValue int64 = 0
		createdNodes[releaseName] = &initialValue
	}
	if err == nil {
		atomic.AddInt64(createdNodes[releaseName], 1)
	}
	return err
}



func CheckNodeCount(t *testing.T, releaseName model.ReleaseName) error {
	result, err := runQuery(t, releaseName, "MATCH (n) RETURN COUNT(n) AS count", noParams)

	if err != nil {
		return err
	}

	if value, found := result[0].Get("count"); found {
		countedNodes := value.(int64)
		assert.Equal(t, atomic.LoadInt64(createdNodes[releaseName]), countedNodes)
		return err
	} else {
		return fmt.Errorf("expected at least one result")
	}
}

func runQuery(t *testing.T, releaseName model.ReleaseName, cypher string, params map[string]interface{}) ([]*neo4j.Record, error) {

	boltPort, cleanupProxy, proxyErr := proxyBolt(t, releaseName)
	defer cleanupProxy()
	if proxyErr != nil {
		return nil, proxyErr
	}

	driver, err := neo4j.NewDriver(fmt.Sprintf("%s:%d", dbUri, boltPort), authToUse, func(config *neo4j.Config) {
	})
	// Handle driver lifetime based on your application lifetime requirements  driver's lifetime is usually
	// bound by the application lifetime, which usually implies one driver instance per application
	defer driver.Close()

	if err := awaitConnectivity(t, err, driver); err != nil {
		return nil, err
	}

	// Sessions are shortlived, cheap to create and NOT thread safe. Typically create one or more sessions
	// per request in your web application. Make sure to call Close on the session when done.
	// For multidatabase support, set sessionConfig.DatabaseName to requested database
	// Session config will default to write mode, if only reads are to be used configure session for
	// read mode.
	session := driver.NewSession(neo4j.SessionConfig{DatabaseName: dbName})
	defer session.Close()

	result, err := session.Run(cypher, params)
	if err != nil {
		return nil, err
	}

	return result.Collect()
}

func awaitConnectivity(t *testing.T, err error, driver neo4j.Driver) error {
	// This polls verify connectivity until it succeeds or it times out. We should be able to remove this when we have readiness probes (maybe)
	start := time.Now()
	timeoutAfter := time.Minute * 3
	for {
		t.Log("Checking connectivity for ", dbUri)
		err = driver.VerifyConnectivity()
		if err == nil {
			t.Log("Connectivity check passed for ", dbUri)
			return nil
		} else if neo4j.IsNeo4jError(err) && strings.Contains(err.(*neo4j.Neo4jError).Code, "CredentialsExpired") {
			t.Logf("recieved CredentialsExpired message from driver. Attempting to proceed")
			return nil
		} else {
			elapsed := time.Now().Sub(start)
			if elapsed > timeoutAfter {
				return err
			} else {
				t.Logf("Connectivity check failed (%s), retrying...", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

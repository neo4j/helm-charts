package integration_tests

import (
	"errors"
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
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

func checkNeo4jConfiguration(t *testing.T, releaseName model.ReleaseName, expectedConfiguration *model.Neo4jConfiguration) (err error) {

	var runtimeConfig []*neo4j.Record
	var expectedOverrides = map[string]string{
		"server.https.enabled":           "true",
		"server.bolt.tls_level":          "REQUIRED",
		"server.directories.logs":        "/logs",
		"server.directories.metrics":     "/metrics",
		"server.directories.import":      "/import",
		"server.panic.shutdown_on_panic": "true",
	}

	deadline := time.Now().Add(3 * time.Minute)
	for true {
		if !time.Now().Before(deadline) {
			msg := fmt.Sprintf("timed out fetching config:  %d", len(runtimeConfig))
			t.Error(msg)
			return errors.New(msg)
		}
		runtimeConfig, err = runQuery(t, releaseName, "CALL dbms.listConfig() YIELD name, value", nil, false)
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
		if name == "server.jvm.additional" {
			assert.Equal(t, expectedConfiguration.JvmArgs(), strings.Split(value, "\n"))
		}
	}

	if err == nil {
		t.Log("Configuration check passed for:", releaseName.String())
	}
	return err
}

func createNode(t *testing.T, releaseName model.ReleaseName) error {
	_, err := runQuery(t, releaseName, "CREATE (n:Item { id: $id, name: $name }) RETURN n.id, n.name", map[string]interface{}{
		"id":   1,
		"name": "Item 1",
	},
		false)
	if _, found := createdNodes[releaseName]; !found {
		var initialValue int64 = 0
		createdNodes[releaseName] = &initialValue
	}
	if err == nil {
		atomic.AddInt64(createdNodes[releaseName], 1)
	}
	return err
}

// createDatabase runs a cypher query to create a database with the given name
func createDatabase(t *testing.T, releaseName model.ReleaseName, databaseName string) error {
	cypherQuery := fmt.Sprintf("CREATE DATABASE %s", databaseName)
	_, err := runQuery(t, releaseName, cypherQuery, nil, false)
	if !assert.NoError(t, err) {
		return fmt.Errorf("error seen while creating database %s , err := %v", databaseName, err)
	}
	//sleep is required so that CheckDataBase is able to fetch the above created database
	//It takes few seconds for the new database to be updated.
	// Do not reduce the time to anything less than 10 , tests would fail
	time.Sleep(10 * time.Second)
	return nil
}

// checkDataBaseExists runs a cypher query to check if the given database name exists or not
func checkDataBaseExists(t *testing.T, releaseName model.ReleaseName, databaseName string) error {
	cypherQuery := fmt.Sprintf("SHOW DATABASE %s YIELD name", databaseName)
	results, err := runQuery(t, releaseName, cypherQuery, nil, false)
	if !assert.NoError(t, err) {
		t.Logf("%v", err)
		return fmt.Errorf("error seen while creating database %s , err := %v", databaseName, err)
	}
	if !assert.NotEqual(t, len(results), 0) {
		return fmt.Errorf("no results received from cypher query")
	}

	for _, result := range results {
		if value, found := result.Get("name"); found {
			if assert.Equal(t, value, databaseName) {
				return nil
			}
		}
	}
	return fmt.Errorf("no record yielded for cypher query %s", cypherQuery)
}

// checkApocConfig fires a apoc cypher query
// It's a way to check if apoc plugin is loaded and the customized apoc config is loaded or not
func checkApocConfig(t *testing.T, releaseName model.ReleaseName) error {

	results, err := runQuery(t, releaseName, "CALL apoc.create.node([\"Person\", \"Actor\"], {name: \"Tom Hanks\"});", nil, false)
	if !assert.NoError(t, err) {
		t.Logf("%v", err)
		return fmt.Errorf("error seen while firing apoc cypher \n err := %v", err)
	}
	if !assert.NotEqual(t, len(results), 0) {
		return fmt.Errorf("no results received from cypher query")
	}
	return nil
}

// checkNodeCount runs the cypher query to get the number of nodes on a cluster core
func checkNodeCount(t *testing.T, releaseName model.ReleaseName) error {
	result, err := runQuery(t, releaseName, "MATCH (n) RETURN COUNT(n) AS count", noParams, false)

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

func runQuery(t *testing.T, releaseName model.ReleaseName, cypher string, params map[string]interface{}, connectToPod bool) ([]*neo4j.Record, error) {

	boltPort, cleanupProxy, proxyErr := proxyBolt(t, releaseName, connectToPod)
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

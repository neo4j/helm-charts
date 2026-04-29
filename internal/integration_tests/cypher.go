package integration_tests

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

// Auth stuff
const dbUri = "neo4j+ssc://localhost"
const boltDbUri = "bolt+ssc://localhost"
const user = "neo4j"
const dbName = "neo4j"
const systemDbName = "system"

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
		"server.directories.import":      "/import",
		"server.panic.shutdown_on_panic": "true",
	}
	if model.Neo4jEdition == "enterprise" {
		expectedOverrides["server.directories.metrics"] = "/metrics"
	}
	deadline := time.Now().Add(3 * time.Minute)
	for true {
		if !time.Now().Before(deadline) {
			msg := fmt.Sprintf("timed out fetching config:  %d", len(runtimeConfig))
			t.Error(msg)
			return errors.New(msg)
		}
		runtimeConfig, err = runQuery(t, releaseName, "CALL dbms.listConfig() YIELD name, value", nil, model.Neo4jEdition == "community")
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
		if valueUntyped == nil {
			valueUntyped = ""
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
		model.Neo4jEdition == "community")
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

// stopDatabase runs a cypher query to stop a database with the given name
func stopDatabase(t *testing.T, releaseName model.ReleaseName, databaseName string) error {
	cypherQuery := fmt.Sprintf("STOP DATABASE %s", databaseName)
	_, err := runQueryViaSystemDB(t, releaseName, cypherQuery, nil, false)
	if !assert.NoError(t, err) {
		return fmt.Errorf("error seen while stopping database %s , err := %v", databaseName, err)
	}
	return nil
}

// startDatabase runs a cypher query to start a database with the given name
func startDatabase(t *testing.T, releaseName model.ReleaseName, databaseName string) error {
	cypherQuery := fmt.Sprintf("START DATABASE %s", databaseName)
	_, err := runQueryViaSystemDB(t, releaseName, cypherQuery, nil, false)
	if !assert.NoError(t, err) {
		return fmt.Errorf("error seen while starting database %s , err := %v", databaseName, err)
	}
	return nil
}

// createMoviesDataSet runs movie dataset cypher query
func createMoviesDataSet(t *testing.T, releaseName model.ReleaseName) error {
	_, err := runQuery(t, releaseName, model.MOVIES_CYPHER, nil, false)
	if !assert.NoError(t, err) {
		return fmt.Errorf("error seen while creating movies dataset, err := %v", err)
	}
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
	result, err := runQuery(t, releaseName, "MATCH (n) RETURN COUNT(n) AS count", noParams, model.Neo4jEdition == "community")

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

// checkLdapPassword runs a cypher query to get ldapPassword and checks if the ldapPassword is set or not
func checkLdapPassword(t *testing.T, releaseName model.ReleaseName) error {
	time.Sleep(30 * time.Second)

	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		result, err := runQuery(t, releaseName,
			"CALL dbms.listConfig('dbms.security.ldap.authorization.system_password') YIELD value",
			noParams,
			model.Neo4jEdition == "community")

		if err != nil {
			if strings.Contains(err.Error(), "NotALeader") {
				t.Logf("Attempt %d: Hit non-leader node, retrying...", i+1)
				time.Sleep(10 * time.Second)
				lastErr = err
				continue
			}
			return err
		}

		value, found := result[0].Get("value")
		if !found {
			return fmt.Errorf("expected at least one result")
		}

		ldapPass := value.(string)
		assert.NotEqual(t, ldapPass, "No Value", "LdapPassword not set !!")
		return nil
	}

	return fmt.Errorf("failed to check LDAP password after %d attempts: %v", maxRetries, lastErr)
}

// checkBloomVersion runs the cypher query to get bloom license info
func checkBloomVersion(t *testing.T, releaseName model.ReleaseName) error {
	result, err := runQuery(t, releaseName, "CALL bloom.checkLicenseCompliance() YIELD status;", noParams, model.Neo4jEdition == "community")
	if err != nil {
		return err
	}

	value, found := result[0].Get("status")
	if !found {
		return fmt.Errorf("expected bloom license status, found nothing !!")
	}
	status := value.(string)
	assert.Equal(t, status, "valid", fmt.Sprintf("bloom license status found %s is not matching with 'valid'", status))
	return nil
}

func runQuery(t *testing.T, releaseName model.ReleaseName, cypher string, params map[string]interface{}, connectToPod bool) ([]*neo4j.Record, error) {

	boltPort, cleanupProxy, proxyErr := proxyBolt(t, releaseName, connectToPod)
	defer cleanupProxy()
	if proxyErr != nil {
		return nil, proxyErr
	}
	ctx := context.Background()
	driver, err := neo4j.NewDriverWithContext(fmt.Sprintf("%s:%d", dbUri, boltPort), authToUse, func(config *neo4j.Config) {
	})
	// Handle driver lifetime based on your application lifetime requirements  driver's lifetime is usually
	// bound by the application lifetime, which usually implies one driver instance per application
	defer driver.Close(ctx)

	if err := awaitConnectivity(t, err, driver, ctx); err != nil {
		return nil, err
	}

	// Sessions are shortlived, cheap to create and NOT thread safe. Typically create one or more sessions
	// per request in your web application. Make sure to call Close on the session when done.
	// For multidatabase support, set sessionConfig.DatabaseName to requested database
	// Session config will default to write mode, if only reads are to be used configure session for
	// read mode.
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: dbName})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}

	return result.Collect(ctx)
}

func runQueryViaSystemDB(t *testing.T, releaseName model.ReleaseName, cypher string, params map[string]interface{}, connectToPod bool) ([]*neo4j.Record, error) {

	boltPort, cleanupProxy, proxyErr := proxyBolt(t, releaseName, connectToPod)
	defer cleanupProxy()
	if proxyErr != nil {
		return nil, proxyErr
	}
	ctx := context.Background()
	driver, err := neo4j.NewDriverWithContext(fmt.Sprintf("%s:%d", boltDbUri, boltPort), authToUse, func(config *neo4j.Config) {
	})
	// Handle driver lifetime based on your application lifetime requirements  driver's lifetime is usually
	// bound by the application lifetime, which usually implies one driver instance per application
	defer driver.Close(ctx)

	if err := awaitConnectivity(t, err, driver, ctx); err != nil {
		return nil, err
	}

	// Sessions are shortlived, cheap to create and NOT thread safe. Typically create one or more sessions
	// per request in your web application. Make sure to call Close on the session when done.
	// For multidatabase support, set sessionConfig.DatabaseName to requested database
	// Session config will default to write mode, if only reads are to be used configure session for
	// read mode.
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: systemDbName})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}

	return result.Collect(ctx)
}

func awaitConnectivity(t *testing.T, err error, driver neo4j.DriverWithContext, ctx context.Context) error {
	// This polls verify connectivity until it succeeds or it times out. We should be able to remove this when we have readiness probes (maybe)
	start := time.Now()
	timeoutAfter := time.Minute * 3
	for {
		t.Log("Checking connectivity for ", dbUri)
		err = driver.VerifyConnectivity(ctx)
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

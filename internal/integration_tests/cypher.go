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
	"github.com/neo4j/helm-charts/internal/testutil/poll"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

// isNotALeader returns true for Neo4j NotALeader errors that surface briefly
// during leader election and are safe to retry.
func isNotALeader(err error) bool {
	if err == nil {
		return false
	}
	var neoErr *neo4j.Neo4jError
	if errors.As(err, &neoErr) {
		return strings.Contains(neoErr.Code, "NotALeader")
	}
	return strings.Contains(err.Error(), "NotALeader")
}

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

var errPortForwardExited = errors.New("port forward exited while waiting for Neo4j connectivity")

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

// createMoviesDataSet runs movie dataset cypher query, retrying on NotALeader errors
// which can occur when the cluster is still electing a leader after previous tests
func createMoviesDataSet(t *testing.T, releaseName model.ReleaseName) error {
	return poll.Until(context.Background(), t, poll.Opts{
		Interval:      10 * time.Second,
		Timeout:       2 * time.Minute,
		Description:   "MOVIES_CYPHER to run against a leader",
		RetryableErrs: isNotALeader,
	}, func(context.Context) (bool, error) {
		_, err := runQuery(t, releaseName, model.MOVIES_CYPHER, nil, false)
		return err == nil, err
	})
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
	return poll.Until(context.Background(), t, poll.Opts{
		Interval:      10 * time.Second,
		Timeout:       30 * time.Second, // 3 × 10s preserves historical budget
		Description:   "APOC create.node to run against a leader",
		RetryableErrs: isNotALeader,
	}, func(context.Context) (bool, error) {
		results, err := runQuery(t, releaseName, "CALL apoc.create.node([\"Person\", \"Actor\"], {name: \"Tom Hanks\"});", nil, false)
		if err != nil {
			return false, err
		}
		if len(results) == 0 {
			return false, fmt.Errorf("no results received from cypher query")
		}
		return true, nil
	})
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

// checkLdapPassword runs a cypher query to get ldapPassword and checks if the ldapPassword is set or not.
// The 60-second timeout absorbs both the post-install config-propagation wait
// (previously a fixed 30s sleep up front) and leader-election retries.
func checkLdapPassword(t *testing.T, releaseName model.ReleaseName) error {
	var ldapPass string
	err := poll.Until(context.Background(), t, poll.Opts{
		Interval:      10 * time.Second,
		Timeout:       60 * time.Second,
		Description:   "LDAP system_password to be observable via listConfig",
		RetryableErrs: isNotALeader,
	}, func(context.Context) (bool, error) {
		result, err := runQuery(t, releaseName,
			"CALL dbms.listConfig('dbms.security.ldap.authorization.system_password') YIELD value",
			noParams,
			model.Neo4jEdition == "community")
		if err != nil {
			return false, err
		}
		value, found := result[0].Get("value")
		if !found {
			return false, fmt.Errorf("no value in listConfig result")
		}
		ldapPass = value.(string)
		return true, nil
	})
	if err != nil {
		return err
	}
	assert.NotEqual(t, ldapPass, "No Value", "LdapPassword not set !!")
	return nil
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
	return runQueryInDatabase(t, releaseName, cypher, params, connectToPod, dbUri, dbName)
}

func runQueryViaSystemDB(t *testing.T, releaseName model.ReleaseName, cypher string, params map[string]interface{}, connectToPod bool) ([]*neo4j.Record, error) {
	return runQueryInDatabase(t, releaseName, cypher, params, connectToPod, boltDbUri, systemDbName)
}

func runQueryInDatabase(t *testing.T, releaseName model.ReleaseName, cypher string, params map[string]interface{}, connectToPod bool, uri string, databaseName string) ([]*neo4j.Record, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		records, err := runQueryInDatabaseOnce(t, releaseName, cypher, params, connectToPod, uri, databaseName)
		if err == nil {
			return records, nil
		}
		lastErr = err
		if !errors.Is(err, errPortForwardExited) {
			return nil, err
		}
		if attempt == 3 {
			break
		}
		t.Logf("retrying Neo4j query after dropped port-forward (attempt %d/3): %v", attempt, err)
		time.Sleep(5 * time.Second)
	}
	return nil, lastErr
}

func runQueryInDatabaseOnce(t *testing.T, releaseName model.ReleaseName, cypher string, params map[string]interface{}, connectToPod bool, uri string, databaseName string) ([]*neo4j.Record, error) {
	boltPort, cleanupProxy, proxyDone, proxyErr := proxyBolt(t, releaseName, connectToPod)
	if cleanupProxy != nil {
		defer cleanupProxy()
	}
	if proxyErr != nil {
		return nil, fmt.Errorf("%w: %v", errPortForwardExited, proxyErr)
	}
	ctx := context.Background()
	driver, err := neo4j.NewDriverWithContext(fmt.Sprintf("%s:%d", uri, boltPort), authToUse, func(config *neo4j.Config) {
	})
	if err != nil {
		return nil, err
	}
	// Handle driver lifetime based on your application lifetime requirements  driver's lifetime is usually
	// bound by the application lifetime, which usually implies one driver instance per application
	defer driver.Close(ctx)

	if err := awaitConnectivity(t, nil, driver, ctx, uri, proxyDone); err != nil {
		return nil, err
	}

	// Sessions are shortlived, cheap to create and NOT thread safe. Typically create one or more sessions
	// per request in your web application. Make sure to call Close on the session when done.
	// For multidatabase support, set sessionConfig.DatabaseName to requested database
	// Session config will default to write mode, if only reads are to be used configure session for
	// read mode.
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: databaseName})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}

	return result.Collect(ctx)
}

func awaitConnectivity(t *testing.T, driverErr error, driver neo4j.DriverWithContext, ctx context.Context, uri string, proxyDone <-chan error) error {
	if driverErr != nil {
		return driverErr
	}
	// Remove this when Neo4j readiness probes gate traffic; until then we poll.
	return poll.Until(ctx, t, poll.Opts{
		Interval:      5 * time.Second,
		Timeout:       3 * time.Minute,
		Description:   "neo4j driver VerifyConnectivity to " + uri,
		RetryableErrs: func(err error) bool { return !errors.Is(err, errPortForwardExited) },
	}, func(pctx context.Context) (bool, error) {
		select {
		case err := <-proxyDone:
			if err == nil {
				return false, errPortForwardExited
			}
			return false, fmt.Errorf("%w: %v", errPortForwardExited, err)
		default:
		}
		err := driver.VerifyConnectivity(pctx)
		if err == nil {
			return true, nil
		}
		select {
		case proxyErr := <-proxyDone:
			if proxyErr == nil {
				return false, errPortForwardExited
			}
			return false, fmt.Errorf("%w: %v", errPortForwardExited, proxyErr)
		default:
		}
		// CredentialsExpired means the server is up and responding — good enough to proceed.
		var neoErr *neo4j.Neo4jError
		if errors.As(err, &neoErr) && strings.Contains(neoErr.Code, "CredentialsExpired") {
			t.Logf("received CredentialsExpired; connectivity is established")
			return true, nil
		}
		return false, err
	})
}

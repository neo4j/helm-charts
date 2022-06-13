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

func CheckNeo4jConfiguration(t *testing.T, releaseName model.ReleaseName, expectedConfiguration *model.Neo4jConfiguration) (err error) {

	var runtimeConfig []*neo4j.Record
	var expectedOverrides = map[string]string{
		"dbms.connector.https.enabled":  "true",
		"dbms.connector.bolt.tls_level": "REQUIRED",
		"dbms.directories.logs":         "/logs",
		"dbms.directories.metrics":      "/metrics",
		"dbms.directories.import":       "/import",
		"dbms.panic.shutdown_on_panic":  "true",
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

//CreateDatabase runs a cypher query to create a database with the given name
func CreateDatabase(t *testing.T, releaseName model.ReleaseName, databaseName string) error {
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

//CheckDataBaseExists runs a cypher query to check if the given database name exists or not
func CheckDataBaseExists(t *testing.T, releaseName model.ReleaseName, databaseName string) error {
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

//CreateNodeOnReadReplica fires a cypher query to create a node
//It's a way to check write requests to read replica are routed to the cluster core since read replica DOES NOT perform writes
func CreateNodeOnReadReplica(t *testing.T, releaseName model.ReleaseName) error {
	_, err := runQuery(t, releaseName, "CREATE (n:Item { id: $id, name: $name }) RETURN n.id, n.name", map[string]interface{}{
		"id":   1,
		"name": "Item 1",
	},
		true)
	if _, found := createdNodes[releaseName]; !found {
		var initialValue int64 = 0
		createdNodes[releaseName] = &initialValue
	}
	if err == nil {
		atomic.AddInt64(createdNodes[releaseName], 1)
	}
	return err
}

//CheckApocConfig fires a apoc cypher query
//It's a way to check if apoc plugin is loaded and the customized apoc config is loaded or not
func CheckApocConfig(t *testing.T, releaseName model.ReleaseName) error {

	results, err := runQuery(t, releaseName, "CALL apoc.config.list() YIELD key, value WHERE key = \"apoc.jdbc.apoctest.url\" RETURN *;", nil, false)
	if !assert.NoError(t, err) {
		t.Logf("%v", err)
		return fmt.Errorf("error seen while firing apoc cypher \n err := %v", err)
	}
	if !assert.NotEqual(t, len(results), 0) {
		return fmt.Errorf("no results received from cypher query")
	}

	for _, result := range results {
		if value, found := result.Get("value"); found {
			if assert.Equal(t, value, "jdbc:foo:bar") {
				return nil
			}
		}
	}
	return fmt.Errorf("no record yielded for apoc cypher query")
}

//CheckReadReplicaConfiguration checks runs a cypher query to check the read replica configuration
// the configuration dbms.mode so retrieved must contain the value READ_REPLICA
func CheckReadReplicaConfiguration(t *testing.T, releaseName model.ReleaseName) error {
	result, err := runReadOnlyQuery(t, releaseName, "CALL dbms.listConfig(\"dbms.mode\") YIELD value", noParams)
	if err != nil {
		return err
	}

	if !assert.Equal(t, len(result), 1) {
		return fmt.Errorf("unexpected results from cypher query")
	}

	if value, found := result[0].Get("value"); found {
		assert.Equal(t, value, "READ_REPLICA")
		return nil
	}

	return fmt.Errorf("unable to get dbms.mode using cypher")
}

//CheckReadReplicaServerGroupsConfiguration checks if the server_groups config contains "read-replicas" or not for read replica cluster
func CheckReadReplicaServerGroupsConfiguration(t *testing.T, releaseName model.ReleaseName) error {
	result, err := runReadOnlyQuery(t, releaseName, "CALL dbms.cluster.overview() YIELD groups", noParams)
	if err != nil {
		return err
	}

	//if result is empty throw error
	if !assert.NotEmpty(t, result) {
		return fmt.Errorf("No results received from cypher query for read replica server groups")
	}

	var readReplicasCount int
	for _, resultRow := range result {
		if rowValues, found := resultRow.Get("groups"); found {
			for _, value := range rowValues.([]interface{}) {
				if groupName, ok := value.(string); ok {
					if groupName == "read-replicas" {
						readReplicasCount++
					}
				}
			}
		}
	}

	if !assert.Equal(t, readReplicasCount, 2) {
		return fmt.Errorf("unable to get any group from dbms.cluster.overview() containing read-replicas using cypher")
	}

	return nil
}

//CheckNodeCount runs the cypher query to get the number of nodes on a cluster core
func CheckNodeCount(t *testing.T, releaseName model.ReleaseName) error {
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

//UpdateReadReplicaConfig updates the read replica upstream strategy on the provided chart
func UpdateReadReplicaConfig(t *testing.T, releaseName model.ReleaseName, extraArgs ...string) error {

	diskName := releaseName.DiskName()
	err := run(
		t,
		"helm",
		model.BaseHelmCommand(
			"upgrade",
			releaseName,
			model.ClusterReadReplicaHelmChart,
			model.Neo4jEdition,
			&diskName,
			append(extraArgs, "--wait", "--timeout", "300s")...,
		)...,
	)
	if !assert.NoError(t, err) {
		return err
	}

	err = run(t, "kubectl", "--namespace", string(releaseName.Namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/"+releaseName.String())
	if !assert.NoError(t, err) {
		return err
	}

	return nil
}

//CheckNodeCountOnReadReplica performs a cypher query to check node count.
//We are not using here createdNodes Map since it does not contain the correct count of nodes retrieved via read replica
func CheckNodeCountOnReadReplica(t *testing.T, releaseName model.ReleaseName, expectedCount int64) error {

	countnodes := func() ([]*neo4j.Record, error) {
		return runQuery(t, releaseName, "MATCH (n) RETURN COUNT(n) AS count", noParams, true)
	}

	timeout := time.After(2 * time.Minute)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout reached while counting nodes on read replica")
		default:
			var result []*neo4j.Record
			var err error
			if result, err = countnodes(); err != nil {
				return fmt.Errorf("Missing count from cypher query %s", err)
			}

			if value, found := result[0].Get("count"); found {
				countedNodes := value.(int64)
				if expectedCount != countedNodes {
					continue
				}
				return nil
			}
			return fmt.Errorf("count key is not received from read replica")
		}
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

//runReadOnlyQuery triggers a read only cypher query used by read replica
func runReadOnlyQuery(t *testing.T, releaseName model.ReleaseName, cypher string, params map[string]interface{}) ([]*neo4j.Record, error) {

	boltPort, cleanupProxy, proxyErr := proxyBolt(t, releaseName, true)
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

	transactionWork := func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(cypher, params)
		if err != nil {
			return nil, err
		}
		return result.Collect()
	}

	result, err := session.ReadTransaction(transactionWork)
	if err != nil {
		return nil, err
	}

	return result.([]*neo4j.Record), nil
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

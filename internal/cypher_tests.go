package internal

import (
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var dbUri = "neo4j://localhost:7687"
var user = "neo4j"
var dbName = "neo4j"

// BEGIN auth stuff

var defaultPassword = "neo4j"
var desiredPassword = fmt.Sprintf("%d", RandomIntBetween(100000, 999999999))

// pointer to the auth token that the driver should use
var authToUse *neo4j.AuthToken = nil

// Mutex for operations on the authToUse pointer
var authLock sync.Mutex

func SetPassword() error {
	authLock.Lock()
	defer authLock.Unlock()

	var dbNameBefore = dbName
	defer func() {
		dbUri = strings.Replace(dbUri, "bolt://", "neo4j://", 1)
		dbName = dbNameBefore
	}()
	dbUri = strings.Replace(dbUri, "neo4j://", "bolt://", 1)
	dbName = "system"

	var auth = neo4j.BasicAuth(user, defaultPassword, "")
	authToUse = &auth
	_, err := runQuery( fmt.Sprintf("ALTER CURRENT USER SET PASSWORD FROM '%s' TO '%s'", defaultPassword, desiredPassword), noParams)
	if err == nil {
		auth = neo4j.BasicAuth(user, desiredPassword, "")
		authToUse = &auth
	} else {
		authToUse = nil
	}
	return err
}

// Check that we have setup driver auth token
func checkAuthSet() bool {
	authLock.Lock()
	defer authLock.Unlock()
	return authToUse != nil
}

// END auth stuff

// Track the total number of nodes that we've created
var createdNodes int64 = 0

// empty param map (makes queries without params more readable)
var noParams = map[string]interface{}{}

func CreateNode() error {
	_, err := runQuery("CREATE (n:Item { id: $id, name: $name }) RETURN n.id, n.name", map[string]interface{}{
		"id":   1,
		"name": "Item 1",
	})
	if err == nil {
		atomic.AddInt64(&createdNodes, 1)
	}
	return err
}

func CheckNodeCount(t *testing.T) error {
	result, err := runQuery("MATCH (n) RETURN COUNT(n) AS count", noParams)

	if err != nil {
		return err
	}

	if value, found := result[0].Get("count"); found {
		countedNodes := value.(int64)
		assert.Equal(t, atomic.LoadInt64(&createdNodes), countedNodes)
		return err
	} else {
		return fmt.Errorf("expected at least one result")
	}
}

func runQuery(cypher string, params map[string]interface{}) ([]*neo4j.Record, error) {
	if authToUse == nil && !checkAuthSet() {
		return nil, fmt.Errorf("driver's auth token has not yet been set")
	}

	driver, err := neo4j.NewDriver(dbUri, *authToUse)
	// Handle driver lifetime based on your application lifetime requirements  driver's lifetime is usually
	// bound by the application lifetime, which usually implies one driver instance per application
	defer driver.Close()

	if err := awaitConnectivity(err, driver); err != nil {
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

func awaitConnectivity(err error, driver neo4j.Driver) error {
	// This polls verify connectivity until it succeeds or it times out. We should be able to remove this when we have readiness probes (maybe)
	start := time.Now()
	timeoutAfter := time.Minute * 3
	for {
		fmt.Print("Checking connectivity for ", dbUri)
		err = driver.VerifyConnectivity()
		if err == nil {
			return nil
		} else if neo4j.IsNeo4jError(err) && strings.Contains(err.(*neo4j.Neo4jError).Code, "CredentialsExpired") {
			log.Printf("recieved CredentialsExpired message from driver. Attempting to proceed")
			return nil
		} else {
			elapsed := time.Now().Sub(start)
			if elapsed > timeoutAfter {
				return err
			} else {
				fmt.Printf("Connectivity check failed (%s), retrying...", err)
				time.Sleep(time.Second)
			}
		}
	}
}


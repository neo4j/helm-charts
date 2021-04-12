package internal

import (
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/stretchr/testify/assert"
	"gopkg.in/ini.v1"
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
var foo int

const neo4jConfJvmAdditionalKey = "dbms.jvm.additional"

func init() {
	foo = 1

}

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
	_, err := runQuery(fmt.Sprintf("ALTER CURRENT USER SET PASSWORD FROM '%s' TO '%s'", defaultPassword, desiredPassword), noParams)
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

type Neo4jConfiguration struct {
	conf    map[string]string
	jvmArgs []string
}

func (c *Neo4jConfiguration) PopulateFromFile(filename string) Neo4jConfiguration {
	yamlFile, err := ini.ShadowLoad(filename)
	CheckError(err)
	defaultSection := yamlFile.Section("")

	jvmAdditional, err := defaultSection.GetKey(neo4jConfJvmAdditionalKey)
	CheckError(err)
	c.jvmArgs = jvmAdditional.StringsWithShadows("\n")
	c.conf = defaultSection.KeysHash()
	delete(c.conf, neo4jConfJvmAdditionalKey)

	return *c
}

func (c *Neo4jConfiguration) Update(other Neo4jConfiguration) Neo4jConfiguration {
	var jvmArgs []string
	if len(other.jvmArgs) > 0 {
		jvmArgs = other.jvmArgs
	} else {
		jvmArgs = c.jvmArgs
	}
	for k, v := range other.conf {
		c.conf[k] = v
	}

	return Neo4jConfiguration{
		jvmArgs: jvmArgs,
		conf: c.conf,
	}
}

// Very quick test to check that no errors are thrown and a couple of values from the default neo4j conf show up
func TestPopulateFromFile(t *testing.T) {

	t.Run("populateFromFile", func(t *testing.T) {
		conf := (&Neo4jConfiguration{}).PopulateFromFile("neo4j/neo4j.conf")
		value, found := conf.conf["dbms.windows_service_name"]
		assert.True(t, found)
		assert.Equal(t, "neo4j", value)

		_, jvmKeyFound := conf.conf[neo4jConfJvmAdditionalKey]
		assert.False(t, jvmKeyFound)

		assert.Contains(t, conf.jvmArgs, "-XX:+UnlockDiagnosticVMOptions")
		assert.Contains(t, conf.jvmArgs, "-XX:+DebugNonSafepoints")
		assert.Greater(t, len(conf.jvmArgs), 1)
	})
}

func CheckNeo4jConfiguration(t *testing.T, expectedConfiguration Neo4jConfiguration) (err error) {

	var runtimeConfig []*neo4j.Record

	deadline := time.Now().Add(3 * time.Minute)
	for true {
		if !time.Now().Before(deadline) {
			return fmt.Errorf("timed out fetching config:  %d", len(runtimeConfig))
		}
		runtimeConfig, err = runQuery("CALL dbms.listConfig() YIELD name, value", nil)
		CheckError(err)
		if len(runtimeConfig) >= len(expectedConfiguration.conf) {
			break
		}
	}

	for _, record := range runtimeConfig {
		nameUntyped, foundName := record.Get("name")
		valueUntyped, foundValue := record.Get("value")
		if !(foundName && foundValue) {
			panic("record is missing expected name or value")
		}

		name := nameUntyped.(string)
		value := valueUntyped.(string)
		if expectedValue, found := expectedConfiguration.conf[name]; found {
			assert.Equal(t, strings.ToLower(expectedValue), strings.ToLower(value),
				"Expected runtime config for %s to match provided value", name)
		}
		if name == "dbms.jvm.additional" {
			assert.Equal(t, expectedConfiguration.jvmArgs, strings.Split(value, "\n"))
		}
	}

	return err
}

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

	cleanupProxy, proxyErr := proxyBolt()
	defer cleanupProxy()
	CheckError(proxyErr)

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

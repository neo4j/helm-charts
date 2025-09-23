package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func ExecuteEnablement(username, pass string) error {
	// serviceName = core-4.default.svc.cluster.local:7687
	serviceName := fmt.Sprintf("%s.%s.svc.cluster.local:7687", ReleaseName, Namespace)
	ctx := context.Background()
	driver, err := getNeo4jDriver(ctx, username, pass)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)
	serverId, serverState, err := getNeo4jServerIdAndState(ctx, driver, serviceName)
	if err != nil {
		return err
	}
	if strings.ToLower(serverState) == "enabled" {
		log.Println("Server is already enabled !!")
		return nil
	}

	err = enableNeo4jServer(ctx, driver, serverId)
	if err != nil {
		return err
	}

	enabled, err := isNeo4jServerEnabled(ctx, driver, serviceName)
	if err != nil {
		return err
	}
	if !enabled {
		return fmt.Errorf("Server is NOT ENABLED !!")
	}

	log.Println("Server is ENABLED !!")
	return nil
}

func getNeo4jDriver(ctx context.Context, username, pass string) (neo4j.DriverWithContext, error) {
	log.Println("Establishing connection with Neo4j")
	retries := 5
	serviceName := fmt.Sprintf("%s.%s.svc.cluster.local:7687", ReleaseName, Namespace)
	// URI examples: "neo4j://localhost", "neo4j+s://xxx.databases.neo4j.io"
	dbUri := fmt.Sprintf("%s://%s", os.Getenv("PROTOCOL"), serviceName)
	dbUser := username
	dbPassword := pass

	// Prepare authentication
	auth := neo4j.BasicAuth(dbUser, dbPassword, "")

	// Check if we need to configure TLS settings
	var configFunc func(*neo4j.Config)
	if isTLSProtocol(os.Getenv("PROTOCOL")) {
		log.Println("TLS protocol detected, checking SSL configuration...")
		tlsConfig, err := createTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %v", err)
		}
		if tlsConfig != nil {
			log.Println("Using custom TLS configuration")
			configFunc = func(config *neo4j.Config) {
				config.TlsConfig = tlsConfig
			}
		}
	}

	var driver neo4j.DriverWithContext
	var err error
	for i := 1; i <= retries; i++ {
		if configFunc != nil {
			driver, err = neo4j.NewDriverWithContext(dbUri, auth, configFunc)
		} else {
			driver, err = neo4j.NewDriverWithContext(dbUri, auth)
		}

		if err != nil {
			log.Printf("failed to create Neo4j driver: %v", err)
			if i == retries {
				return nil, err
			}
			log.Printf("sleeping for 30 seconds. Retry (%d/%d)\n", i, retries)
			time.Sleep(30 * time.Second)
			continue
		}

		err = driver.VerifyConnectivity(ctx)
		if err != nil && i == retries {
			return nil, err
		}
		if err != nil {
			log.Printf("found error while trying to connect to Neo4j (db uri := %s)\n %s", dbUri, err.Error())
			log.Printf("sleeping for 30 seconds. Retry (%d/%d)\n", i, retries)
			time.Sleep(30 * time.Second)
			continue
		}
		break
	}
	log.Println("Connectivity established !!")
	return driver, nil
}

// isTLSProtocol checks if the protocol uses TLS
func isTLSProtocol(protocol string) bool {
	return strings.Contains(protocol, "+s")
}

// createTLSConfig creates a TLS configuration based on environment variables
func createTLSConfig() (*tls.Config, error) {
	disableHostnameVerification := false
	insecureSkipVerify := false

	// Parse SSL_DISABLE_HOSTNAME_VERIFICATION
	if val := os.Getenv("SSL_DISABLE_HOSTNAME_VERIFICATION"); val != "" {
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			log.Printf("Warning: Invalid SSL_DISABLE_HOSTNAME_VERIFICATION value '%s', using false", val)
		} else {
			disableHostnameVerification = parsed
		}
	}

	// Parse SSL_INSECURE_SKIP_VERIFY
	if val := os.Getenv("SSL_INSECURE_SKIP_VERIFY"); val != "" {
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			log.Printf("Warning: Invalid SSL_INSECURE_SKIP_VERIFY value '%s', using false", val)
		} else {
			insecureSkipVerify = parsed
		}
	}

	// If no custom TLS settings are needed, return nil to use defaults
	if !disableHostnameVerification && !insecureSkipVerify {
		return nil, nil
	}

	tlsConfig := &tls.Config{}

	if insecureSkipVerify {
		log.Println("WARNING: TLS certificate verification is disabled. This is not recommended for production use.")
		tlsConfig.InsecureSkipVerify = true
	} else if disableHostnameVerification {
		log.Println("TLS hostname verification is disabled")
		tlsConfig.InsecureSkipVerify = false
		// For hostname verification only, create a custom VerifyPeerCertificate function that verifies the certificate chain but not the hostname
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return nil
		}
	}

	return tlsConfig, nil
}

// enableNeo4jServer fires the cypher query ENABLE SERVER <server-id>
func enableNeo4jServer(ctx context.Context, driver neo4j.DriverWithContext, serverId string) error {

	log.Println("Enabling server")
	// Enable SERVER 'serverId'
	query := fmt.Sprintf("ENABLE SERVER \"%s\"", serverId)
	_, err := neo4j.ExecuteQuery(ctx, driver,
		query,
		nil, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("system"))
	if err != nil {
		return err
	}

	return nil
}

// getNeo4jServerIdAndState returns the id and state for the provided serviceName from the list of records
func getNeo4jServerIdAndState(ctx context.Context, driver neo4j.DriverWithContext, serviceName string) (string, string, error) {
	log.Println("Fetching Neo4j server id and state")
	var serverState, serverId string
	query := fmt.Sprintf("SHOW SERVERS WHERE address='%s'", serviceName)
	result, err := neo4j.ExecuteQuery(ctx, driver,
		query,
		nil, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("system"))
	if err != nil {
		return serverId, serverState, err
	}

	if len(result.Records) != 1 {
		return serverId, serverState, fmt.Errorf("more than one or no records found for address %s \n records %v", serviceName, result.Records)
	}

	serverIdAny, present := result.Records[0].Get("name")
	if !present {
		return serverId, serverState, fmt.Errorf("'name' key not present in record")
	}
	serverId = serverIdAny.(string)

	serverStateAny, present := result.Records[0].Get("state")
	if !present {
		return serverId, serverState, fmt.Errorf("'state' key not present in record")
	}
	serverState = serverStateAny.(string)

	if serverId == "" || serverState == "" {
		return "", "", fmt.Errorf("cannot find serverId and serverState for %s", serviceName)
	}
	return serverId, serverState, nil
}

// isNeo4jServerEnabled checks whether the server
func isNeo4jServerEnabled(ctx context.Context, driver neo4j.DriverWithContext, serviceName string) (bool, error) {

	_, serverState, err := getNeo4jServerIdAndState(ctx, driver, serviceName)
	if err != nil {
		return false, err
	}
	if strings.ToLower(serverState) != "enabled" {
		return false, nil
	}
	return true, nil
}

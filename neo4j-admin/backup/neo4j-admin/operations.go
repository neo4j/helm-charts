package neo4j_admin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"google.golang.org/api/iterator"
)

// CheckDatabaseConnectivity checks if there is connectivity with the provided backup instance or not
func CheckDatabaseConnectivity(hostPortList string) error {
	// Split by comma to handle multiple endpoints
	endpoints := strings.Split(hostPortList, ",")

	var lastErr error
	for _, endpoint := range endpoints {
		endpoint = strings.TrimSpace(endpoint)
		address := strings.Split(endpoint, ":")

		if len(address) != 2 {
			lastErr = fmt.Errorf("invalid endpoint format %s, expected host:port", endpoint)
			log.Printf("Warning: %v", lastErr)
			continue
		}

		output, err := exec.Command("nc", "-vz", address[0], address[1]).CombinedOutput()
		if err != nil {
			lastErr = fmt.Errorf("connectivity cannot be established with %s \n output = %s \n err = %v",
				endpoint, string(output), err)
			log.Printf("Warning: %v", lastErr)
			continue
		}

		outputString := strings.ToLower(string(output))
		if !strings.Contains(outputString, "succeeded") && !strings.Contains(outputString, "connected") {
			lastErr = fmt.Errorf("connectivity cannot be established with %s. Missing 'succeeded' in output \n output = %s",
				endpoint, string(output))
			log.Printf("Warning: %v", lastErr)
			continue
		}

		log.Printf("Connectivity established with Database %s!!", endpoint)
		return nil // Return on first successful connection
	}

	// If we get here, all endpoints failed
	return fmt.Errorf("connectivity cannot be established with any endpoint: %v", lastErr)
}

// importCustomCA imports the custom S3 CA certificate into a JVM truststore and updates system CA certificates
// if S3_CA_CERT_PATH is set. This should now support both JSSE-based clients (via JKS truststore) and CRT-based clients
func importCustomCA() error {
	caPath := os.Getenv("S3_CA_CERT_PATH")
	if caPath == "" {
		log.Println("No custom S3 CA certificate path set, skipping truststore import.")
		return nil
	}

	if err := updateSystemCACertificates(caPath); err != nil {
		log.Printf("Warning: Failed to update system CA certificates (CRT clients may fail): %v", err)
	}

	truststorePath := "/tmp/custom-truststore.jks"
	truststorePass := os.Getenv("TRUSTSTORE_PASSWORD")
	if truststorePass == "" {
		truststorePass = "changeit" // Default for backward compatibility
	}

	// Check if keytool is available
	if _, err := exec.LookPath("keytool"); err != nil {
		log.Printf("Warning: keytool not found in PATH, skipping JKS truststore import: %v", err)
		// Continue - system certs may be sufficient for CRT clients
	} else {
		// Import the CA certificate into the truststore
		cmd := exec.Command("keytool",
			"-importcert",
			"-noprompt",
			"-trustcacerts",
			"-alias", "custom-s3-ca",
			"-file", caPath,
			"-keystore", truststorePath,
			"-storepass", truststorePass,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Warning: Failed to import CA certificate into JKS truststore: %v\noutput: %s", err, string(output))
		} else {
			log.Printf("Successfully imported CA certificate from %s into truststore %s", caPath, truststorePath)

			// Set JAVA_OPTS to use the custom truststore
			javaOpts := fmt.Sprintf("-Djavax.net.ssl.trustStore=%s -Djavax.net.ssl.trustStorePassword=%s", truststorePath, truststorePass)
			if existingOpts := os.Getenv("JAVA_OPTS"); existingOpts != "" {
				javaOpts = existingOpts + " " + javaOpts
			}
			if err := os.Setenv("JAVA_OPTS", javaOpts); err != nil {
				log.Printf("Warning: Failed to set JAVA_OPTS: %v", err)
			} else {
				log.Printf("Set JAVA_OPTS: %s", javaOpts)
			}
		}
	}

	return nil
}

// updateSystemCACertificates copies the custom CA certificate to the system certificate directory
// and runs update-ca-certificates to make it available to CRT-based clients.
func updateSystemCACertificates(caPath string) error {
	// Check if the CA file exists
	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		return fmt.Errorf("CA certificate file not found at %s", caPath)
	}

	// Check if update-ca-certificates is available
	if _, err := exec.LookPath("update-ca-certificates"); err != nil {
		return fmt.Errorf("update-ca-certificates not found in PATH (required for CRT client support): %v", err)
	}

	// Copy CA certificate to system certificate directory
	systemCertPath := "/usr/local/share/ca-certificates/custom-s3-ca.crt"
	sourceFile, err := os.Open(caPath)
	if err != nil {
		return fmt.Errorf("failed to open CA certificate file: %v", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(systemCertPath)
	if err != nil {
		return fmt.Errorf("failed to create system certificate file (may require root permissions): %v", err)
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy CA certificate: %v", err)
	}

	// Run update-ca-certificates to update the system trust store
	cmd := exec.Command("update-ca-certificates")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update system CA certificates: %v\noutput: %s", err, string(output))
	}

	log.Printf("Successfully updated system CA certificates with custom S3 CA from %s", caPath)
	log.Printf("update-ca-certificates output: %s", string(output))
	return nil
}

// PerformBackup performs the backup operation and returns the generated backup file name
func PerformBackup(address string) ([]string, error) {
	// Import custom CA if needed
	if err := importCustomCA(); err != nil {
		return nil, fmt.Errorf("failed to import custom CA: %v", err)
	}

	databases := strings.ReplaceAll(os.Getenv("DATABASE"), ",", " ")
	flags := getBackupCommandFlags(address)
	log.Printf("Printing backup flags %v", flags)
	dir, _ := os.Getwd()
	log.Println("current directory", dir)

	cmd := exec.Command("neo4j-admin", flags...)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Failed to start backup command: %v", err)
	}

	var outputBuffer strings.Builder

	stdoutDone := make(chan bool)
	stderrDone := make(chan bool)

	// Start goroutine to read and stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println(line)
			outputBuffer.WriteString(line + "\n")
		}
		stdoutDone <- true
	}()

	// Start goroutine to read and stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println(line)
			outputBuffer.WriteString(line + "\n")
		}
		stderrDone <- true
	}()

	// Wait for both stdout and stderr to be fully read
	<-stdoutDone
	<-stderrDone

	// Wait for the command to complete
	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("Backup Failed for database %s !! output = %s \n err = %v", databases, outputBuffer.String(), err)
	}

	log.Printf("Backup completed successfully for database %s", databases)
	backupFileNames, err := retrieveBackupFileNames(outputBuffer.String())
	if err != nil {
		return nil, err
	}
	return backupFileNames, nil
}

// PerformConsistencyCheck performs the consistency check on the backup taken and returns the generated report tar name
func PerformConsistencyCheck(database string, backupFileName string) (string, error) {
	// Use the provided backup file name (without .backup extension if present)
	fileName := strings.TrimSuffix(backupFileName, ".backup")
	if fileName == "" {
		return "", fmt.Errorf("backup file name cannot be empty for consistency check")
	}

	// Ensure temp directory exists for cloud storage consistency checks
	cloudProvider := os.Getenv("CLOUD_PROVIDER")
	if cloudProvider != "" {
		tempPath := os.Getenv("CONSISTENCY_CHECK_TEMP_DIR")
		if tempPath == "" {
			tempPath = filepath.Join(getBackupPath(), "consistency-temp")
		}
		if err := os.MkdirAll(tempPath, 0755); err != nil {
			log.Printf("Warning: Failed to create temp directory %s: %v", tempPath, err)
		} else {
			log.Printf("Created consistency check temp directory: %s", tempPath)
		}

		// For AWS, verify that required environment variables are set
		if cloudProvider == "aws" {
			awsRegion := os.Getenv("AWS_REGION")
			awsDefaultRegion := os.Getenv("AWS_DEFAULT_REGION")
			awsCredsFile := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
			bucketName := os.Getenv("BUCKET_NAME")

			log.Printf("AWS configuration for consistency check:")
			log.Printf("  AWS_REGION: %s", awsRegion)
			log.Printf("  AWS_DEFAULT_REGION: %s", awsDefaultRegion)
			log.Printf("  AWS_SHARED_CREDENTIALS_FILE: %s", awsCredsFile)
			log.Printf("  BUCKET_NAME: %s", bucketName)

			if awsRegion == "" && awsDefaultRegion == "" {
				log.Printf("Warning: No AWS region configured, this might cause consistency check to fail")
			}
			if awsCredsFile == "" {
				log.Printf("Warning: No AWS credentials file configured, this might cause consistency check to fail")
			}
		}
	}

	flags := getConsistencyCheckCommandFlags(fileName, database)
	log.Printf("Printing consistency check flags %v", flags)
	log.Printf("Individual flags:")
	for i, flag := range flags {
		log.Printf("  [%d]: %s", i, flag)
	}

	log.Printf("Starting consistency check execution for database %s", database)
	log.Printf("Backup file name: %s", fileName)
	log.Printf("Cloud provider: %s", cloudProvider)

	// Set timeout for consistency checks - configurable via CONSISTENCY_CHECK_TIMEOUT environment variable
	// Empty string means no timeout (for local storage)
	var ctx context.Context
	var cancel context.CancelFunc
	var timeoutDuration time.Duration

	timeoutEnv := os.Getenv("CONSISTENCY_CHECK_TIMEOUT")
	if timeoutEnv != "" {
		// Parse and apply timeout
		if parsedTimeout, err := time.ParseDuration(timeoutEnv); err != nil {
			// Invalid format - use default 30m for cloud, no timeout for local
			if cloudProvider != "" {
				timeoutDuration = 30 * time.Minute
				log.Printf("Warning: Invalid timeout format '%s', using default 30m: %v", timeoutEnv, err)
				ctx, cancel = context.WithTimeout(context.Background(), timeoutDuration)
			} else {
				log.Printf("Warning: Invalid timeout format '%s', no timeout applied for local storage: %v", timeoutEnv, err)
				ctx, cancel = context.WithCancel(context.Background())
			}
		} else {
			// Valid timeout provided
			timeoutDuration = parsedTimeout
			log.Printf("Using configured timeout of %v for consistency check", timeoutDuration)
			ctx, cancel = context.WithTimeout(context.Background(), timeoutDuration)
		}
	} else {
		// No timeout specified - use default for cloud, none for local
		if cloudProvider != "" {
			timeoutDuration = 30 * time.Minute
			log.Printf("Using default extended timeout of %v for cloud storage consistency check", timeoutDuration)
			ctx, cancel = context.WithTimeout(context.Background(), timeoutDuration)
		} else {
			log.Printf("No timeout applied for local storage consistency check")
			ctx, cancel = context.WithCancel(context.Background())
		}
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, "neo4j-admin", flags...)

	// Log the exact command being executed
	log.Printf("Executing command: neo4j-admin %s", strings.Join(flags, " "))
	log.Printf("Working directory: %s", getBackupPath())

	// Create pipes for stdout and stderr to get real-time output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("Failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("Failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("Failed to start consistency check command: %v", err)
	}

	var outputBuffer strings.Builder
	stdoutDone := make(chan bool)
	stderrDone := make(chan bool)

	// Start goroutine to read and stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("CONSISTENCY_CHECK_STDOUT: %s", line)
			outputBuffer.WriteString(line + "\n")
		}
		log.Printf("CONSISTENCY_CHECK: stdout reading completed")
		stdoutDone <- true
	}()

	// Start goroutine to read and stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("CONSISTENCY_CHECK_STDERR: %s", line)
			outputBuffer.WriteString(line + "\n")
		}
		log.Printf("CONSISTENCY_CHECK: stderr reading completed")
		stderrDone <- true
	}()

	// Wait for both stdout and stderr to be fully read
	log.Printf("CONSISTENCY_CHECK: waiting for stdout and stderr to complete")
	<-stdoutDone
	<-stderrDone
	log.Printf("CONSISTENCY_CHECK: stdout and stderr reading completed")

	// Wait for the command to complete
	log.Printf("CONSISTENCY_CHECK: waiting for command to complete")
	err = cmd.Wait()
	log.Printf("CONSISTENCY_CHECK: command completed with error: %v", err)

	log.Printf("Consistency check command completed. Output length: %d bytes", len(outputBuffer.String()))
	log.Printf("Consistency check output: %s", outputBuffer.String())

	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("Consistency check timed out after %v for database %s", timeoutDuration, database)
	}

	if err == nil {
		log.Printf("No inconsistencies found for database %s !! No Inconsistency report generated.", database)
		return "", nil
	}

	var me *exec.ExitError
	if errors.As(err, &me) {
		log.Printf("Inconsistencies found for database %s. Exit code was %d\n", database, me.ExitCode())
		log.Printf("Consistency Check Completed !!")

		tarFileName := fmt.Sprintf("%s/%s.report.tar.gz", getBackupPath(), fileName)
		directoryName := fmt.Sprintf("%s/%s.report", getBackupPath(), fileName)
		log.Printf("tarfileName %s directoryName %s", tarFileName, directoryName)
		_, err = exec.Command("tar", "-czvf", tarFileName, directoryName, "--absolute-names").CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("Unable to create a tar archive of consistency check report for database %s !! \n output = %s \n err = %v", database, outputBuffer.String(), err)
		}
		log.Printf("Consistency Check Report tar archive created for database %s at %s !!", database, tarFileName)
		return fmt.Sprintf("%s.report.tar.gz", fileName), nil
	}
	return "", fmt.Errorf("Consistency Check Failed for database %s!! \n output = %s \n err = %v", database, outputBuffer.String(), err)
}

// New functions to count backups

func getBackupCount(db string, fromPath string) (int, error) {
	u, err := url.Parse(fromPath)
	if err != nil {
		return 0, err
	}

	// Trim leading and trailing slashes from path - count functions add their own separator
	// e.g. "/backups/" -> "backups" to avoid "backups//neo4j-" prefix that wouldn't match "backups/neo4j-*.backup"
	pathPrefix := strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")

	switch u.Scheme {
	case "s3":
		return countS3Backups(db, u.Host, pathPrefix)
	case "gs":
		return countGCSBackups(db, u.Host, pathPrefix)
	case "azb":
		return countAzureBackups(db, u.Host, pathPrefix)
	default:
		// local path
		return countLocalBackups(fromPath, db)
	}
}

func countLocalBackups(path, db string) (int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}
	count := 0
	prefix := db + "-"
	suffix := ".backup"
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), suffix) {
			count++
		}
	}
	return count, nil
}

func countS3Backups(db, bucket, pathPrefix string) (int, error) {
	config := &aws.Config{}

	// Check for custom S3 endpoint from environment variable
	if endpoint := os.Getenv("AWS_ENDPOINT_URL_S3"); endpoint != "" {
		config.Endpoint = aws.String(endpoint)
		log.Printf("Using custom S3 endpoint: %s", endpoint)
	}

	// Set S3 force path style if configured
	if forcePathStyle := os.Getenv("S3_FORCE_PATH_STYLE"); forcePathStyle == "true" {
		config.S3ForcePathStyle = aws.Bool(true)
		log.Printf("Using S3 force path style")
	}

	// Set region if configured
	if region := os.Getenv("AWS_REGION"); region != "" {
		config.Region = aws.String(region)
		log.Printf("Using AWS region: %s", region)
	} else if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		config.Region = aws.String(region)
		log.Printf("Using AWS default region: %s", region)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return 0, err
	}
	client := s3.New(sess)
	prefix := pathPrefix
	if prefix != "" {
		prefix += "/"
	}
	prefix += db + "-"
	count := 0
	err = client.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			if strings.HasSuffix(*obj.Key, ".backup") {
				count++
			}
		}
		return true
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func countGCSBackups(db, bucket, pathPrefix string) (int, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Close()
	prefix := pathPrefix
	if prefix != "" {
		prefix += "/"
	}
	prefix += db + "-"
	it := client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	count := 0
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		if strings.HasSuffix(obj.Name, ".backup") {
			count++
		}
	}
	return count, nil
}

func countAzureBackups(db, account, containerPath string) (int, error) {
	storageAccount := os.Getenv("AZURE_STORAGE_ACCOUNT")
	if storageAccount == "" {
		return 0, fmt.Errorf("AZURE_STORAGE_ACCOUNT environment variable is required")
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return 0, err
	}
	// Use AZURE_BLOB_SERVICE_URL if provided (for GovCloud support), otherwise default to commercial Azure
	blobServiceURL := os.Getenv("AZURE_BLOB_SERVICE_URL")
	if blobServiceURL == "" {
		blobServiceURL = "blob.core.windows.net"
	}
	serviceURL := fmt.Sprintf("https://%s.%s", storageAccount, blobServiceURL)
	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return 0, err
	}
	container := containerPath
	if strings.Contains(container, "/") {
		// If path has /, it's account/container, but since account is in host, container is path
		container = strings.Trim(container, "/")
	} else {
		container = account // if no path, host is container
	}
	prefix := db + "-"
	pager := client.NewListBlobsFlatPager(container, &azblob.ListBlobsFlatOptions{
		Prefix: &prefix,
	})
	count := 0
	for pager.More() {
		resp, err := pager.NextPage(context.TODO())
		if err != nil {
			return 0, err
		}
		for _, blob := range resp.Segment.BlobItems {
			if strings.HasSuffix(*blob.Name, ".backup") {
				count++
			}
		}
	}
	return count, nil
}

// Modify PerformAggregateBackup to check backup count
func PerformAggregateBackup() error {
	databaseStr := os.Getenv("AGGREGATE_BACKUP_DATABASE")
	databases := strings.Split(databaseStr, ",")
	for i, db := range databases {
		databases[i] = strings.TrimSpace(db)
	}
	log.Printf("Performing aggregate backups for databases: %s", strings.Join(databases, ", "))

	// Get fromPath
	fromPath := os.Getenv("AGGREGATE_BACKUP_FROM_PATH")
	if fromPath == "" {
		cloudProvider := os.Getenv("CLOUD_PROVIDER")
		if cloudProvider != "" {
			fromPath = getCloudStoragePath()
		} else {
			fromPath = getBackupPath()
		}
	}

	for _, db := range databases {
		if db == "" {
			continue
		}

		// For wildcards, skip the backup count check and let neo4j-admin handle database discovery
		if db == "*" {
			log.Printf("Wildcard '*' detected, passing to neo4j-admin for native wildcard handling")
		} else {
			// For explicit database names, check if aggregation is needed
			count, err := getBackupCount(db, fromPath)
			if err != nil {
				return fmt.Errorf("failed to check backup count for database %s: %v", db, err)
			}
			if count <= 1 {
				log.Printf("Skipping aggregation for database %s as there are only %d backups (no chain to aggregate)", db, count)
				continue
			}
		}

		flags := GetAggregateBackupCommandFlags(db)
		log.Printf("Printing aggregate backup flags for %s: %v", db, flags)
		dir, _ := os.Getwd()
		log.Println("current directory", dir)

		// Log important environment variables for debugging
		log.Printf("Environment variables for S3 access:")
		log.Printf("  CLOUD_PROVIDER: %s", os.Getenv("CLOUD_PROVIDER"))
		log.Printf("  AWS_REGION: %s", os.Getenv("AWS_REGION"))
		log.Printf("  AWS_DEFAULT_REGION: %s", os.Getenv("AWS_DEFAULT_REGION"))
		log.Printf("  AWS_SHARED_CREDENTIALS_FILE: %s", os.Getenv("AWS_SHARED_CREDENTIALS_FILE"))
		log.Printf("  AWS_ENDPOINT_URL_S3: %s", os.Getenv("AWS_ENDPOINT_URL_S3"))
		log.Printf("  S3_CA_CERT_PATH: %s", os.Getenv("S3_CA_CERT_PATH"))
		log.Printf("  S3_SKIP_VERIFY: %s", os.Getenv("S3_SKIP_VERIFY"))
		log.Printf("  S3_FORCE_PATH_STYLE: %s", os.Getenv("S3_FORCE_PATH_STYLE"))
		log.Printf("  S3_SIGNATURE_VERSION: %s", os.Getenv("S3_SIGNATURE_VERSION"))
		log.Printf("  AWS_REQUEST_CHECKSUM_CALCULATION: %s", os.Getenv("AWS_REQUEST_CHECKSUM_CALCULATION"))
		log.Printf("  AWS_RESPONSE_CHECKSUM_VALIDATION: %s", os.Getenv("AWS_RESPONSE_CHECKSUM_VALIDATION"))
		log.Printf("  AWS_S3_DISABLE_MULTIPART_CHECKSUMS: %s", os.Getenv("AWS_S3_DISABLE_MULTIPART_CHECKSUMS"))
		log.Printf("  AGGREGATE_BACKUP_FROM_PATH: %s", os.Getenv("AGGREGATE_BACKUP_FROM_PATH"))
		cmd := exec.Command("neo4j-admin", flags...)

		log.Printf("Executing command line for %s: %s %s", db, cmd.Path, strings.Join(cmd.Args[1:], " "))

		// Create pipes for stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("Failed to create stdout pipe: %v", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("Failed to create stderr pipe: %v", err)
		}

		os.Setenv("JAVA_OPTS", "--add-opens=java.base/java.nio=ALL-UNNAMED")

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("Failed to start aggregate backup command: %v", err)
		}

		// Create a buffer to store the complete output for parsing later
		var outputBuffer strings.Builder

		// Create channels to signal when reading is done
		stdoutDone := make(chan bool)
		stderrDone := make(chan bool)

		// Start goroutine to read and stream stdout
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				log.Println(line)
				outputBuffer.WriteString(line + "\n")
			}
			stdoutDone <- true
		}()

		// Start goroutine to read and stream stderr
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				log.Println(line)
				outputBuffer.WriteString(line + "\n")
			}
			stderrDone <- true
		}()

		// Wait for both stdout and stderr to be fully read
		<-stdoutDone
		<-stderrDone

		// Wait for the command to complete
		err = cmd.Wait()
		output := outputBuffer.String()
		noAggregationNeeded := strings.Contains(strings.ToLower(output), "no aggregation needed") ||
			strings.Contains(strings.ToLower(output), "no need to aggregate")
		if err != nil {
			if noAggregationNeeded {
				log.Printf("No aggregation needed for database %s, treating as success", db)
				log.Printf("%s", output)
				continue
			}
			return fmt.Errorf("Aggregate Backup Failed for database %s !! output = %s \n err = %v", db, output, err)
		}
		log.Printf("Aggregate backup completed successfully for database %s", db)
		if !noAggregationNeeded {
			backupFileNames, err := retrieveAggregatedBackupFileNames(output)
			if err != nil {
				return err
			}
			log.Printf("%s", backupFileNames)
		}
		log.Printf("%s", output)
	}
	return nil
}

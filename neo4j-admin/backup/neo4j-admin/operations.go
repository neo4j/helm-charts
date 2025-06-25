package neo4j_admin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

// PerformBackup performs the backup operation and returns the generated backup file name
func PerformBackup(address string) ([]string, error) {
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

	// Increase timeout to 30 minutes for cloud storage consistency checks
	timeout := 30 * time.Minute
	if cloudProvider != "" {
		log.Printf("Using extended timeout of %v for cloud storage consistency check", timeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
		return "", fmt.Errorf("Consistency check timed out after %v for database %s", timeout, database)
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

// PerformAggregateBackup triggers the neo4j-admin aggregate backup command
func PerformAggregateBackup() error {
	flags := GetAggregateBackupCommandFlags()
	database := os.Getenv("AGGREGATE_BACKUP_DATABASE")
	log.Printf("Printing aggregate backup flags %v", flags)
	dir, _ := os.Getwd()
	log.Println("current directory", dir)

	cmd := exec.Command("neo4j-admin", flags...)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failed to create stderr pipe: %v", err)
	}

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
	if err != nil {
		return fmt.Errorf("Aggregate Backup Failed for database %s !! output = %s \n err = %v", database, outputBuffer.String(), err)
	}

	log.Printf("Aggregate backup completed successfully for database %s", database)
	if !strings.Contains(outputBuffer.String(), "no need to aggregate") {
		backupFileNames, err := retrieveAggregatedBackupFileNames(outputBuffer.String())
		if err != nil {
			return err
		}
		log.Printf("%s", backupFileNames)
	}
	log.Printf(outputBuffer.String())
	return nil
}

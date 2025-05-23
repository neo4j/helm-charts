package neo4j_admin

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
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

	log.Printf("Backup Completed for database %s !!", databases)
	backupFileNames, err := retrieveBackupFileNames(outputBuffer.String())
	if err != nil {
		return nil, err
	}
	return backupFileNames, nil
}

// PerformConsistencyCheck performs the consistency check on the backup taken and returns the generated report tar name
func PerformConsistencyCheck(database string) (string, error) {
	timeStamp := time.Now().Format("2006-01-02T15-04-05")
	fileName := fmt.Sprintf("%s-%s.backup", database, timeStamp)
	flags := getConsistencyCheckCommandFlags(fileName, database)
	log.Printf("Printing consistency check flags %v", flags)
	output, err := exec.Command("neo4j-admin", flags...).CombinedOutput()
	if err == nil {
		log.Printf("No inconsistencies found for %s database !! No Inconsistency report generated.", database)
		return "", nil
	}

	var me *exec.ExitError
	if errors.As(err, &me) {
		log.Printf("Inconsistencies found for %s database. Exit code was %d\n", database, me.ExitCode())
		log.Printf("Consistency Check Completed !!")

		tarFileName := fmt.Sprintf("/backups/%s.report.tar.gz", fileName)
		directoryName := fmt.Sprintf("/backups/%s.report", fileName)
		log.Printf("tarfileName %s directoryName %s", tarFileName, directoryName)
		_, err = exec.Command("tar", "-czvf", tarFileName, directoryName, "--absolute-names").CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("Unable to create a tar archive of consistency check report for database %s !! \n output = %s \n err = %v", database, string(output), err)
		}
		log.Printf("Consistency Check Report tar archive created for database %s at %s !!", database, tarFileName)
		return fmt.Sprintf("%s.report.tar.gz", fileName), nil
	}
	return "", fmt.Errorf("Consistency Check Failed for database %s!! \n output = %s \n err = %v", database, string(output), err)
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

	log.Printf("Aggregate Backup Completed for database %s !!", database)
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

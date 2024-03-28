package neo4j_admin

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CheckDatabaseConnectivity checks if there is connectivity with the provided backup instance or not
func CheckDatabaseConnectivity(hostPort string) error {
	address := strings.Split(hostPort, ":")
	output, err := exec.Command("nc", "-vz", address[0], address[1]).CombinedOutput()
	if err != nil {
		return fmt.Errorf("connectivity cannot be established \n output = %s \n err = %v", string(output), err)
	}
	outputString := strings.ToLower(string(output))
	if !strings.Contains(outputString, "succeeded") && !strings.Contains(outputString, "connected") {
		return fmt.Errorf("connectivity cannot be established. Missing 'succeeded' in output \n output = %s", string(output))
	}
	log.Printf("Connectivity established with Database %s!!", hostPort)
	return nil
}

// PerformBackup performs the backup operation and returns the generated backup file name
func PerformBackup(address string) ([]string, error) {

	databases := strings.ReplaceAll(os.Getenv("DATABASE"), ",", " ")
	flags := getBackupCommandFlags(address)
	log.Printf("Printing backup flags %v", flags)
	output, err := exec.Command("neo4j-admin", flags...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Backup Failed for database %s !! output = %s \n err = %v", databases, string(output), err)
	}
	log.Printf("Backup Completed for database %s !!", databases)
	backupFileNames, err := retrieveBackupFileNames(string(output))
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

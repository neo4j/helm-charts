package neo4j_admin

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// CheckDatabaseConnectivity checks if there is connectivity with the provided backup instance or not
func CheckDatabaseConnectivity(hostPort string) error {
	address := strings.Split(hostPort, ":")
	output, err := exec.Command("nc", "-vz", address[0], address[1]).CombinedOutput()
	if err != nil {
		return fmt.Errorf("connectivity cannot be established \n output = %s \n err = %v", string(output), err)
	}
	if !strings.Contains(string(output), "succeeded") {
		return fmt.Errorf("connectivity cannot be established. Missing 'succeeded' in output \n output = %s", string(output))
	}
	log.Printf("Connectivity established with Database %s!!", hostPort)
	return nil
}

// PerformBackup performs the backup operation and returns the generated backup file name
func PerformBackup() (string, error) {
	flags := getBackupCommandFlags()
	log.Printf("Printing backup flags %v", flags)
	output, err := exec.Command("neo4j-admin", flags...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Backup Failed !! output = %s \n err = %v", string(output), err)
	}
	log.Printf("Backup Completed !!")
	fileName, err := retrieveBackupFileName(string(output))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(fileName), nil
}

// PerformConsistencyCheck performs the consistency check on the backup taken and returns the generated report tar name
func PerformConsistencyCheck(backupFileName string) (string, error) {
	flags := getConsistencyCheckCommandFlags(backupFileName)
	log.Printf("Printing consistency check flags %v", flags)
	output, err := exec.Command("neo4j-admin", flags...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Consistency Check Failed !! \n output = %s \n err = %v", string(output), err)
	}
	log.Printf("Consistency Check Completed. Report Name %s !!", string(output))

	tarFileName := fmt.Sprintf("/backups/%s.report.tar.gz", backupFileName)
	directoryName := fmt.Sprintf("/backups/%s.report", backupFileName)
	log.Printf("tarfileName %s directoryName %s", tarFileName, directoryName)
	output, err = exec.Command("tar", "-czvf", tarFileName, directoryName, "--absolute-names").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Unable to create a tar archive of consistency check report !! \n output = %s \n err = %v", string(output), err)
	}
	log.Printf("Consistency Check Report tar archive created at %s !!", tarFileName)

	return fmt.Sprintf("%s.report.tar.gz", backupFileName), nil
}

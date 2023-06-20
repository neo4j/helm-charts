package neo4j_admin

import (
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
	if !strings.Contains(string(output), "succeeded") {
		return fmt.Errorf("connectivity cannot be established. Missing 'succeeded' in output \n output = %s", string(output))
	}
	log.Printf("Connectivity established with Database %s!!", hostPort)
	return nil
}

// PerformBackup performs the backup operation and returns the generated backup file name
func PerformBackup(address string) ([]string, error) {
	flags := getBackupCommandFlags(address)
	log.Printf("Printing backup flags %v", flags)
	output, err := exec.Command("neo4j-admin", flags...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Backup Failed !! output = %s \n err = %v", string(output), err)
	}
	log.Printf("Backup Completed !!")
	dbNames := []string{os.Getenv("DATABASE")}
	// if the database contains a "*" parse the output to get the list of databases whose backup is taken
	if strings.Contains(os.Getenv("DATABASE"), "*") || strings.Contains(os.Getenv("DATABASE"), "?") {
		dbNames, err = retrieveBackedUpDBNames(string(output))
		if err != nil {
			return nil, err
		}
	}

	fileNames, err := createTarsForGeneratedBackup(dbNames)
	if err != nil {
		return nil, err
	}

	return fileNames, nil
}

func createTarsForGeneratedBackup(dbNames []string) ([]string, error) {
	var fileNames []string
	for _, dbName := range dbNames {
		timeStamp := time.Now().Format("2006-01-02T15:04:05")
		tarFileName := fmt.Sprintf("%s-%s.backup.tar.gz", dbName, timeStamp)
		tarFilePath := fmt.Sprintf("/backups/%s", tarFileName)
		output, err := exec.Command("tar", "-czvf", tarFilePath, fmt.Sprintf("/backups/%s", dbName), "--absolute-names").CombinedOutput()
		if err != nil {
			log.Printf("tarFilePath %s dbName %s", tarFilePath, dbName)
			return nil, fmt.Errorf("Unable to create a tar archive of backup generated !! \n output = %s \n err = %v", string(output), err)
		}
		fileNames = append(fileNames, tarFileName)
		log.Printf("Backup tar archive created at %s !!", tarFilePath)
	}
	return fileNames, nil
}

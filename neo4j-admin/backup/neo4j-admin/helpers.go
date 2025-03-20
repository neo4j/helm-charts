package neo4j_admin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// getBackupPath returns the configured backup path or the default /backups
func getBackupPath() string {
	if path := os.Getenv("BACKUP_PATH"); path != "" {
		return path
	}
	return "/backups"
}

// getBackupCommandFlags returns a slice of string containing all the flags to be passed with the neo4j-admin backup command
func getBackupCommandFlags(address string) []string {
	flags := []string{"database", "backup"}

	// Split address into multiple endpoints if comma-separated
	endpoints := strings.Split(address, ",")
	for _, endpoint := range endpoints {
		flags = append(flags, fmt.Sprintf("--from=%s", strings.TrimSpace(endpoint)))
	}

	flags = append(flags, fmt.Sprintf("--include-metadata=%s", os.Getenv("INCLUDE_METADATA")))
	flags = append(flags, fmt.Sprintf("--keep-failed=%s", os.Getenv("KEEP_FAILED")))
	flags = append(flags, fmt.Sprintf("--parallel-recovery=%s", os.Getenv("PARALLEL_RECOVERY")))
	flags = append(flags, fmt.Sprintf("--type=%s", os.Getenv("TYPE")))
	flags = append(flags, fmt.Sprintf("--to-path=%s", getBackupPath()))

	if len(strings.TrimSpace(os.Getenv("PAGE_CACHE"))) > 0 {
		flags = append(flags, fmt.Sprintf("--pagecache=%s", os.Getenv("PAGE_CACHE")))
	}

	if os.Getenv("VERBOSE") == "true" {
		flags = append(flags, "--verbose")
	}

	for _, db := range strings.Split(os.Getenv("DATABASE"), ",") {
		flags = append(flags, strings.TrimSpace(db))
	}

	return flags
}

// GetAggregateBackupCommandFlags returns a slice of string containing all the flags to be passed with the neo4j-admin aggregate backup command
func GetAggregateBackupCommandFlags() []string {
	database := os.Getenv("AGGREGATE_BACKUP_DATABASE")
	flags := []string{"backup", "aggregate"}

	// Check for specific aggregate backup temp dir first
	if aggregateTempDir := os.Getenv("AGGREGATE_BACKUP_TEMP_DIR"); aggregateTempDir != "" {
		flags = append(flags, fmt.Sprintf("--temp-path=%s", aggregateTempDir))
	} else {
		// Fall back to using the backup directory
		aggregateTempDir := filepath.Join(getBackupPath(), "aggregate-temp")
		flags = append(flags, fmt.Sprintf("--temp-path=%s", aggregateTempDir))
	}

	flags = append(flags, fmt.Sprintf("--from-path=%s", os.Getenv("AGGREGATE_BACKUP_FROM_PATH")))
	flags = append(flags, fmt.Sprintf("--keep-old-backup=%s", os.Getenv("AGGREGATE_BACKUP_KEEPOLDBACKUP")))
	flags = append(flags, fmt.Sprintf("--parallel-recovery=%s", os.Getenv("AGGREGATE_BACKUP_PARALLEL_RECOVERY")))

	//flags = append(flags, "--expand-commands")
	if os.Getenv("VERBOSE") == "true" {
		flags = append(flags, "--verbose")
	}
	for _, db := range strings.Split(database, ",") {
		flags = append(flags, fmt.Sprintf("%s", db))
	}
	return flags
}

// getConsistencyCheckCommandFlags returns a slice of string containing all the flags to be passed with the neo4j-admin consistency check command
//
//	enable: true
//	checkIndexes: true
//	checkGraph: true
//	checkCounts: true
//	checkPropertyOwners: true
//	maxOffHeapMemory: ""
//	threads: ""
//	verbose: true
func getConsistencyCheckCommandFlags(fileName string, database string) []string {
	flags := []string{"database", "check"}
	backupPath := getBackupPath()

	flags = append(flags, fmt.Sprintf("--check-indexes=%s", os.Getenv("CONSISTENCY_CHECK_INDEXES")))
	flags = append(flags, fmt.Sprintf("--check-graph=%s", os.Getenv("CONSISTENCY_CHECK_GRAPH")))
	flags = append(flags, fmt.Sprintf("--check-counts=%s", os.Getenv("CONSISTENCY_CHECK_COUNTS")))
	flags = append(flags, fmt.Sprintf("--check-property-owners=%s", os.Getenv("CONSISTENCY_CHECK_PROPERTYOWNERS")))
	flags = append(flags, fmt.Sprintf("--report-path=%s/%s.report", backupPath, fileName))
	flags = append(flags, fmt.Sprintf("--from-path=%s", backupPath))
	if len(strings.TrimSpace(os.Getenv("CONSISTENCY_CHECK_THREADS"))) > 0 {
		flags = append(flags, fmt.Sprintf("--threads=%s", os.Getenv("CONSISTENCY_CHECK_THREADS")))
	}
	if len(strings.TrimSpace(os.Getenv("CONSISTENCY_CHECK_MAXOFFHEAPMEMORY"))) > 0 {
		flags = append(flags, fmt.Sprintf("--max-off-heap-memory=%s", os.Getenv("CONSISTENCY_CHECK_MAXOFFHEAPMEMORY")))
	}
	if os.Getenv("CONSISTENCY_CHECK_VERBOSE") == "true" {
		flags = append(flags, "--verbose")
	}
	//flags = append(flags, "--expand-commands")
	flags = append(flags, database)

	return flags
}

// retrieveBackupFileNames takes the backup command output and looks for the below string and retrieves the backup file names
// Ex: Finished artifact creation 'neo4j-2023-05-04T17-21-27.backup' for database 'neo4j', took 121ms.
func retrieveBackupFileNames(cmdOutput string) ([]string, error) {
	re := regexp.MustCompile(`Finished artifact creation (.*).backup`)
	matches := re.FindAllStringSubmatch(cmdOutput, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("regex failed !! cannot retrieve backup file name \n %v", matches)
	}
	var backupFileNames []string
	for _, match := range matches {
		name := strings.Replace(match[1], "'", "", -1)
		backupFileNames = append(backupFileNames, fmt.Sprintf("%s.backup", name))
	}
	return backupFileNames, nil
}

// retrieveAggregatedBackupFileNames takes the output of aggregate backup command and returns the list of succesfully backup chain statements
// Ex: Successfully aggregated backup chain of database 'neo4j2', new artifact: '/var/lib/neo4j/bin/backup/neo4j2-2024-06-13T12-43-43.backup'.
func retrieveAggregatedBackupFileNames(cmdOutput string) ([]string, error) {
	re := regexp.MustCompile(`Successfully aggregated backup chain(.*)`)
	matches := re.FindAllString(cmdOutput, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("regex failed !! cannot retrieve aggregated backup file name \n %v \n %s", matches, cmdOutput)
	}
	return matches, nil
}

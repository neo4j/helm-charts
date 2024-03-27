package neo4j_admin

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// getBackupCommandFlags returns a slice of string containing all the flags to be passed with the neo4j-admin backup command
func getBackupCommandFlags(address string) []string {
	database := os.Getenv("DATABASE")
	flags := []string{"database", "backup"}
	flags = append(flags, fmt.Sprintf("--from=%s", address))
	flags = append(flags, fmt.Sprintf("--include-metadata=%s", os.Getenv("INCLUDE_METADATA")))
	flags = append(flags, fmt.Sprintf("--keep-failed=%s", os.Getenv("KEEP_FAILED")))
	flags = append(flags, fmt.Sprintf("--parallel-recovery=%s", os.Getenv("PARALLEL_RECOVERY")))
	flags = append(flags, fmt.Sprintf("--type=%s", os.Getenv("TYPE")))
	flags = append(flags, fmt.Sprintf("--to-path=%s", "/backups"))
	if len(strings.TrimSpace(os.Getenv("PAGE_CACHE"))) > 0 {
		flags = append(flags, fmt.Sprintf("--pagecache=%s", os.Getenv("PAGE_CACHE")))
	}
	//flags = append(flags, "--expand-commands")
	if os.Getenv("VERBOSE") == "true" {
		flags = append(flags, "--verbose")
	}
	// "neo4j,system,test1" --> 'neo4j' 'system' 'test1'
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

	flags = append(flags, fmt.Sprintf("--check-indexes=%s", os.Getenv("CONSISTENCY_CHECK_INDEXES")))
	flags = append(flags, fmt.Sprintf("--check-graph=%s", os.Getenv("CONSISTENCY_CHECK_GRAPH")))
	flags = append(flags, fmt.Sprintf("--check-counts=%s", os.Getenv("CONSISTENCY_CHECK_COUNTS")))
	flags = append(flags, fmt.Sprintf("--check-property-owners=%s", os.Getenv("CONSISTENCY_CHECK_PROPERTYOWNERS")))
	flags = append(flags, fmt.Sprintf("--report-path=/backups/%s.report", fileName))
	flags = append(flags, fmt.Sprintf("--from-path=/backups"))
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
	if !(len(matches) > 1) {
		return nil, fmt.Errorf("regex failed !! cannot retrieve backup file name \n %v", matches)
	}
	var backupFileNames []string
	for _, match := range matches {
		name := strings.Replace(match[1], "'", "", -1)
		backupFileNames = append(backupFileNames, fmt.Sprintf("%s.backup", name))
	}
	return backupFileNames, nil
}

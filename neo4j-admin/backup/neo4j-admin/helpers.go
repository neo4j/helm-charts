package neo4j_admin

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

// getBackupCommandFlags returns a slice of string containing all the flags to be passed with the neo4j-admin backup command
// fallbackToFull: true
// heapSize: ""
// checkConsistency: true
// checkIndexes: true
// checkIndexStructure: true
// checkGraph: true
// prepareRestore: true
// includeMetadata: "all"
// parallelRecovery: false
// pageCache: ""
// verbose: true
func getBackupCommandFlags(address string) []string {
	flags := []string{"backup"}
	flags = append(flags, fmt.Sprintf("--from=%s", address))
	flags = append(flags, fmt.Sprintf("--fallback-to-full=%s", os.Getenv("FALLBACK_TO_FULL")))
	flags = append(flags, fmt.Sprintf("--include-metadata=%s", os.Getenv("INCLUDE_METADATA")))
	flags = append(flags, fmt.Sprintf("--check-consistency=%s", os.Getenv("CONSISTENCY_CHECK_ENABLE")))
	flags = append(flags, fmt.Sprintf("--check-indexes=%s", os.Getenv("CONSISTENCY_CHECK_INDEXES")))
	flags = append(flags, fmt.Sprintf("--check-index-structure=%s", os.Getenv("CONSISTENCY_CHECK_INDEX_STRUCTURE")))
	flags = append(flags, fmt.Sprintf("--check-graph=%s", os.Getenv("CONSISTENCY_CHECK_GRAPH")))
	flags = append(flags, fmt.Sprintf("--prepare-restore=%s", os.Getenv("PREPARE_RESTORE")))
	flags = append(flags, fmt.Sprintf("--parallel-recovery=%s", os.Getenv("PARALLEL_RECOVERY")))
	flags = append(flags, fmt.Sprintf("--backup-dir=%s", "/backups"))
	flags = append(flags, fmt.Sprintf("--report-dir=%s", "/backups"))
	flags = append(flags, fmt.Sprintf("--database=%s", os.Getenv("DATABASE")))
	if len(strings.TrimSpace(os.Getenv("PAGE_CACHE"))) > 0 {
		flags = append(flags, fmt.Sprintf("--pagecache=%s", os.Getenv("PAGE_CACHE")))
	}
	//flags = append(flags, "--expand-commands")
	if os.Getenv("VERBOSE") == "true" {
		flags = append(flags, "--verbose")
	}

	return flags
}

// retrieveBackedUpDBNames takes the backup command output and looks for the below string and retrieves the backup file paths
// [c.n.b.v.OnlineBackupExecutor] databaseName=never, backupStatus=successful, reason=
// [c.n.b.v.OnlineBackupExecutor] databaseName=neo4j, backupStatus=successful, reason=
// returns [/backups/never,/backups/neo4j]
func retrieveBackedUpDBNames(cmdOutput string) ([]string, error) {
	log.Printf("command output = %v", cmdOutput)
	var dbNames []string
	re := regexp.MustCompile(`databaseName=(.*) backupStatus=successful`)
	matches := re.FindAllStringSubmatch(cmdOutput, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("regex failed !! cannot retrieve backup file name \n %v", matches)
	}
	for _, match := range matches {
		dbName := strings.Replace(match[1], ",", "", -1)
		dbNames = append(dbNames, fmt.Sprintf("%s", dbName))
	}
	return dbNames, nil
}

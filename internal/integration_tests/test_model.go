package integration_tests

import (
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
	"strings"
	"time"
)

var (
	TestRunIdentifier string
)

var Neo4jConfFile = fmt.Sprintf("neo4j-standalone/neo4j-%s.conf", model.Neo4jEdition)

func init() {
	dt := time.Now()
	dateTag := dt.Format("15:04:05 Mon")
	dateTag = strings.ReplaceAll(dateTag, " ", "-")
	dateTag = strings.ReplaceAll(dateTag, ":", "-")
	TestRunIdentifier = strings.ToLower(dateTag)
}

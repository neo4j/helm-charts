package integration_tests

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/neo4j/helm-charts/internal/model"
)

var (
	TestRunIdentifier string
)

var Neo4jConfFile = fmt.Sprintf("neo4j/neo4j-%s.conf", model.Neo4jEdition)

func init() {
	dt := time.Now()
	// Include microseconds and random component for uniqueness
	dateTag := dt.Format("15:04:05.000000 Mon")
	// Add a small random component to ensure uniqueness even within microseconds
	randomSuffix := rand.Intn(1000)
	dateTag = fmt.Sprintf("%s-%d", dateTag, randomSuffix)
	dateTag = strings.ReplaceAll(dateTag, " ", "-")
	dateTag = strings.ReplaceAll(dateTag, ":", "-")
	dateTag = strings.ReplaceAll(dateTag, ".", "-")
	TestRunIdentifier = strings.ToLower(dateTag)
}

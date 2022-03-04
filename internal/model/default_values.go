package model

import (
	"fmt"
	. "github.com/neo4j/helm-charts/internal/helpers"
)

var DefaultPassword = fmt.Sprintf("a%da", RandomIntBetween(100000, 999999999))

const StorageSize = "10Gi"
const cpuRequests = "50m"
const memoryRequests = "900Mi"
const cpuLimits = "1500m"
const memoryLimits = "900Mi"

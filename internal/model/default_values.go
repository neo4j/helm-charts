package model

import (
	"fmt"
	. "github.com/neo4j/helm-charts/internal/helpers"
)

var DefaultPassword = fmt.Sprintf("defaulthelmpassword%da", RandomIntBetween(100000, 999999999))

const StorageSize = "10Gi"
const cpuRequests = "500m"
const memoryRequests = "2Gi"
const cpuLimits = "1500m"
const memoryLimits = "2Gi"

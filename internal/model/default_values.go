package model

import (
	"fmt"
	. "neo4j.com/helm-charts-tests/internal/helpers"
)

var DefaultPassword = fmt.Sprintf("defaulthelmpassword%da", RandomIntBetween(100000, 999999999))

const StorageSize = "10Gi"
const cpuRequests = "50m"
const memoryRequests = "500Mi"
const cpuLimits = "1500m"
const memoryLimits = "900Mi"

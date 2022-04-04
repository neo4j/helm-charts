package aws

import (
	"log"
	"os"
)

type Zone string
type Region string
type ContainerCluster string

func CurrentZone() Zone {
	return Zone(getRequired("AWS_AZ"))
}
func CurrentRegion() Region {
	return Region(getRequired("AWS_DEFAULT_REGION"))
}
func CurrentCluster() ContainerCluster {
	return ContainerCluster(getRequired("AWS_CLUSTER_NAME"))
}

func getRequired(envKey string) string {
	value, found := os.LookupEnv(envKey)
	if !found {
		log.Panicf("Environment variable %s is required but was not set", envKey)
	}
	return value
}

func init() {
	os.Setenv("AWS_CLI_AUTO_PROMPT", "on-partial")
}

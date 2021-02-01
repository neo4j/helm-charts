package internal

import (
	"log"
	"os"
)

type Project string
type Zone string
type Region string
type ContainerCluster string

var CurrentProject Project = Project(getRequired("CLOUDSDK_CORE_PROJECT"))
var CurrentZone Zone = Zone(getRequired("CLOUDSDK_COMPUTE_ZONE"))
var CurrentRegion Region = Region(getRequired("CLOUDSDK_COMPUTE_REGION"))
var CurrentCluster ContainerCluster = ContainerCluster(getRequired("CLOUDSDK_CONTAINER_CLUSTER"))

func getRequired(envKey string) string {
	value, found := os.LookupEnv(envKey)
	if !found {
		log.Panicf("Environment variable %s is required but was not set", envKey)
	}
	return value
}
func init() {
	os.Setenv("CLOUDSDK_CORE_DISABLE_PROMPTS", "True")
}

package model

import (
	"fmt"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"k8s.io/utils/env"
	"log"
	"os"
)

var DefaultPassword = fmt.Sprintf("defaulthelmpassword%da", RandomIntBetween(100000, 999999999))

var ImagePullSecretUsername,
	ImagePullSecretPass,
	ImagePullSecretCustomImageName,
	ImagePullSecretEmail string

var NodeSelectorArgs, ImagePullSecretArgs, CustomApocImageArgs []string

var NodeSelectorLabel = "testLabel=1"

func init() {
	setWorkingDir()
	os.Setenv("KUBECONFIG", ".kube/config")
	ImagePullSecretUsername = env.GetString("IPS_USERNAME", "")
	if ImagePullSecretUsername == "" {
		log.Panic("Please set IPS_USERNAME env variable !!")
	}
	ImagePullSecretPass = env.GetString("IPS_PASS", "")
	if ImagePullSecretPass == "" {
		log.Panic("Please set IPS_PASS env variable !!")
	}
	ImagePullSecretEmail = env.GetString("IPS_EMAIL", "")
	if ImagePullSecretEmail == "" {
		log.Panic("Please set IPS_EMAIL env variable !!")
	}
	ImagePullSecretCustomImageName = env.GetString("NEO4J_DOCKER_IMG", "")
	if ImagePullSecretCustomImageName == "" {
		log.Panic("Please set NEO4J_DOCKER_IMG env variable !!")
	}
	ImagePullSecretArgs = []string{
		"--set", fmt.Sprintf("image.customImage=%s", ImagePullSecretCustomImageName),
		"--set", "image.imagePullSecrets[0]=demo",
		"--set", "image.imageCredentials[0].registry=eu.gcr.io",
		"--set", "image.imageCredentials[0].name=demo",
		"--set", fmt.Sprintf("image.imageCredentials[0].username=%s", ImagePullSecretUsername),
		"--set", fmt.Sprintf("image.imageCredentials[0].password=%s", ImagePullSecretPass),
		"--set", fmt.Sprintf("image.imageCredentials[0].email=%s", ImagePullSecretEmail),
	}
	NodeSelectorArgs = []string{
		"--set", fmt.Sprintf("nodeSelector.%s", NodeSelectorLabel),
	}
	CustomApocImageArgs = []string{
		"--set", fmt.Sprintf("image.customImage=%s", "harshitsinghvi22/apoc:v1"),
	}
}

const StorageSize = "10Gi"
const cpuRequests = "500m"
const memoryRequests = "2Gi"
const cpuLimits = "1500m"
const memoryLimits = "2Gi"

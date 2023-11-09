package model

import (
	"fmt"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"k8s.io/utils/env"
	"log"
	"os"
	"strconv"
)

var DefaultPassword = fmt.Sprintf("defaulthelmpassword%da", RandomIntBetween(100000, 999999999))
var DefaultAuthSecretName = "neo4j-auth"

var ImagePullSecretUsername,
	ImagePullSecretPass,
	ImagePullSecretCustomImageName,
	ImagePullSecretEmail string

var NodeSelectorArgs, ImagePullSecretArgs, PriorityClassNameArgs []string

var NodeSelectorLabel = "testLabel=1"
var DefaultNeo4jName = "test-cluster"
var DefaultNeo4jChartName = "neo4j"
var DefaultNeo4jBackupChartName = "neo4j-admin"
var DefaultNeo4jReverseProxyChartName = "neo4j-reverse-proxy"
var BucketName = "helm-backup-test"
var DefaultClusterSize = 3
var DefaultNeo4jNameArg = []string{"--set", "neo4j.name=" + DefaultNeo4jName}
var LdapArgs = []string{"--set", "ldapPasswordFromSecret=ldapsecret", "--set", "ldapPasswordMountPath=/config/ldapPassword/"}
var DefaultClusterSizeArg = []string{"--set", "neo4j.minimumClusterSize=" + strconv.Itoa(DefaultClusterSize)}
var DefaultEnterpriseSizeArgs = []string{"--set", "neo4j.edition=enterprise", "--set", "neo4j.acceptLicenseAgreement=yes"}

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

	_, present := os.LookupEnv("NEO4J_DOCKER_BACKUP_IMG")
	if !present {
		log.Panic("Please set NEO4J_DOCKER_BACKUP_IMG env variable !!")
	}

	_, present = os.LookupEnv("NEO4J_REVERSE_PROXY_IMG")
	if !present {
		log.Panic("Please set NEO4J_REVERSE_PROXY_IMG env variable !!")
	}

	_, present = os.LookupEnv("BLOOM_LICENSE")
	if !present {
		log.Panic("Please set BLOOM_LICENSE env variable !!")
	}

	_, present = os.LookupEnv("AWS_ACCESS_KEY_ID")
	if !present {
		log.Panic("Please set AWS_ACCESS_KEY_ID env variable !!")
	}

	_, present = os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	if !present {
		log.Panic("Please set AWS_SECRET_ACCESS_KEY env variable !!")
	}

	_, present = os.LookupEnv("AZURE_STORAGE_ACCOUNT_NAME")
	if !present {
		log.Panic("Please set AZURE_STORAGE_ACCOUNT_NAME env variable !!")
	}

	_, present = os.LookupEnv("AZURE_STORAGE_ACCOUNT_KEY")
	if !present {
		log.Panic("Please set AZURE_STORAGE_ACCOUNT_KEY env variable !!")
	}

	_, present = os.LookupEnv("GCP_SERVICE_ACCOUNT_CRED")
	if !present {
		log.Panic("Please set GCP_SERVICE_ACCOUNT_CRED env variable !!. This environment variable holds the json credentials of GCP service account")
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

	PriorityClassNameArgs = []string{
		"--set", fmt.Sprintf("podSpec.priorityClassName=%s", PriorityClassName),
	}
}

const StorageSize = "10Gi"
const cpuRequests = "500m"
const memoryRequests = "2Gi"
const cpuLimits = "1500m"
const memoryLimits = "2Gi"
const PriorityClassName = "high-priority"

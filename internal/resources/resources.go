package resources

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

var _, thisFile, _, _ = runtime.Caller(0)
var resourcesDir = path.Dir(thisFile)
var TestAntiAffinityRule = newYamlFile("testAntiAffinityRule.yaml")
var PluginsInitContainer = newYamlFile("pluginsInitContainer.yaml")
var AcceptLicenseAgreementBoolYes = newYamlFile("acceptLicenseAgreementBoolYes.yaml")
var AcceptLicenseAgreementBoolTrue = newYamlFile("acceptLicenseAgreementBoolTrue.yaml")
var AcceptLicenseAgreement = newYamlFile("acceptLicenseAgreement.yaml")
var ApocCorePlugin = newYamlFile("apocCorePlugin.yaml")
var CsvMetrics = newYamlFile("csvMetrics.yaml")
var DefaultStorageClass = newYamlFile("defaultStorageClass.yaml")
var JvmAdditionalSettings = newYamlFile("jvmAdditionalSettings.yaml")
var BoolsInConfig = newYamlFile("boolsInConfig.yaml")
var IntsInConfig = newYamlFile("intsInConfig.yaml")
var ChmodInitContainer = newYamlFile("chmodInitContainer.yaml")
var ChmodInitContainerAndCustomInitContainer = newYamlFile("chmodInitContainerAndCustomInitContainer.yaml")
var ReadReplicaUpstreamStrategy = newYamlFile("read_replica_upstream_selection_strategy.yaml")
var ExcludeLoadBalancer = newYamlFile("excludeLoadBalancer.yaml")
var EmptyImageCredentials = newYamlFile("imagePullSecret/emptyImageCreds.yaml")
var DuplicateImageCredentials = newYamlFile("imagePullSecret/duplicateImageCreds.yaml")
var MissingImageCredentials = newYamlFile("imagePullSecret/missingImageCreds.yaml")
var EmptyImagePullSecrets = newYamlFile("imagePullSecret/emptyImagePullSecrets.yaml")
var InvalidNodeSelectorLabels = newYamlFile("nodeselector.yaml")

type YamlFile interface {
	Path() string
	HelmArgs() []string
}

type yamlFile struct {
	path string
}

func (y *yamlFile) Path() string {
	return y.path
}

func (y *yamlFile) HelmArgs() []string {
	return []string{"-f", y.path}
}

func resourceExistsAt(path string) (bool, error) {
	if fileInfo, err := os.Stat(path); err == nil {
		if filepath.Ext(path) == ".yaml" && !fileInfo.IsDir() {
			return true, nil
		}
		return false, fmt.Errorf("unexpected error occured. File %s returned fileInfo: %v", path, fileInfo)
	} else {
		return false, err
	}
}

func newYamlFile(filename string) YamlFile {
	fullPath := path.Join(resourcesDir, filename)
	if exists, err := resourceExistsAt(fullPath); err != nil || !exists {
		panic(err)
	}
	return &yamlFile{fullPath}
}

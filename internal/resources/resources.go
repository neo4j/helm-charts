package resources

import (
	"fmt"
	"os"
	"path/filepath"
)

var TestAntiAffinityRule = newYamlFile("./internal/resources/testAntiAffinityRule.yaml")
var PluginsInitContainer = newYamlFile("./internal/resources/pluginsInitContainer.yaml")
var AcceptLicenseAgreementBoolYes = newYamlFile("internal/resources/acceptLicenseAgreementBoolYes.yaml")
var AcceptLicenseAgreementBoolTrue = newYamlFile("internal/resources/acceptLicenseAgreementBoolTrue.yaml")
var AcceptLicenseAgreement = newYamlFile("internal/resources/acceptLicenseAgreement.yaml")
var ApocCorePlugin = newYamlFile("internal/resources/apocCorePlugin.yaml")
var CsvMetrics = newYamlFile("internal/resources/csvMetrics.yaml")
var DefaultStorageClass = newYamlFile("internal/resources/defaultStorageClass.yaml")
var JvmAdditionalSettings = newYamlFile("internal/resources/jvmAdditionalSettings.yaml")
var BoolsInConfig = newYamlFile("internal/resources/boolsInConfig.yaml")
var IntsInConfig = newYamlFile("internal/resources/intsInConfig.yaml")
var ChmodInitContainer = newYamlFile("internal/resources/chmodInitContainer.yaml")
var ChmodInitContainerAndCustomInitContainer = newYamlFile("internal/resources/chmodInitContainerAndCustomInitContainer.yaml")

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

func newYamlFile(path string) YamlFile {
	if exists, err := resourceExistsAt(path); err != nil || !exists {
		panic(err)
	}
	return &yamlFile{path}
}

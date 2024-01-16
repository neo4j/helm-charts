package unit_tests

import (
	"errors"
	"fmt"
	"github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	pv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var acceptLicenseAgreement = []string{"--set", "neo4j.acceptLicenseAgreement=yes"}
var requiredDataMode = []string{"--set", "volumes.data.mode=selector"}
var useDataModeAndAcceptLicense = append(requiredDataMode, acceptLicenseAgreement...)
var readReplicaTesting = []string{"--set", "testing=true"}
var useEnterprise = []string{"--set", "neo4j.edition=enterprise"}
var useCommunity = []string{"--set", "neo4j.edition=community"}
var useEnterpriseAndAcceptLicense = append(useEnterprise, acceptLicenseAgreement...)

// forEachPrimaryChart runs the given test on each helm chart that represents a Neo4j "Primary instance".
// Primary instances are Standalone instances, the primary instance in a Primary+Read Replica(s) configuration and Neo4j Causal Cluster Cores.
// n.b. forEachPrimaryChart runs the tests in parallel.
func forEachPrimaryChart(t *testing.T, test func(*testing.T, model.Neo4jHelmChartBuilder)) {
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder) {
		t.Parallel()
		test(t, chart)
	}

	for _, chart := range model.PrimaryHelmCharts {
		t.Run(t.Name()+chart.Name(), func(t *testing.T) {
			doTestCase(t, chart)
		})
	}
}

// forEachSupportedEdition runs the given test on each Neo4j edition supported by the provided Helm Chart.
// Neo4j editions are "community" and "enterprise". Some helm charts support multiple editions (e.g. neo4j-standalone) and others only support one edition
// (e.g. neo4j-cluster-core only supports Neo4j enterprise edition)
// n.b. forEachSupportedEdition runs the tests in parallel.
func forEachSupportedEdition(t *testing.T, chart model.Neo4jHelmChartBuilder, test func(*testing.T, model.Neo4jHelmChartBuilder, string)) {
	doTestCase := func(t *testing.T, edition string) {
		t.Parallel()
		test(t, chart, edition)
	}

	for _, edition := range chart.GetEditions() {
		t.Run(t.Name()+edition, func(t *testing.T) {
			doTestCase(t, edition)
		})
	}
}

func andEachSupportedEdition(test func(*testing.T, model.Neo4jHelmChartBuilder, string)) func(t *testing.T, chart model.Neo4jHelmChartBuilder) {
	return func(t *testing.T, chart model.Neo4jHelmChartBuilder) {
		forEachSupportedEdition(t, chart, func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
			test(t, chart, edition)
		})
	}
}

func TestErrorThrownIfNoDataVolumeModeChosen(t *testing.T) {
	t.Parallel()
	forEachPrimaryChart(t, andEachSupportedEdition(
		func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
			var helmTemplateArgs []string
			if edition == "enterprise" {
				helmTemplateArgs = acceptLicenseAgreement
			}
			_, err := model.HelmTemplate(t, chart, helmTemplateArgs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "A volume mode for the Neo4j 'data' volume is required.")
			assert.Contains(t, err.Error(), "--set volumes.data.mode=defaultStorageClass")

		}))
}

func TestErrorThrownIfNoVolumeSizeChosen(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(
		func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
			helmArgs := []string{}
			helmArgs = append(helmArgs, requiredDataMode...)
			if edition == "enterprise" {
				helmArgs = append(helmArgs, acceptLicenseAgreement...)
			}

			dynamicLogsVolume := []string{"--set", "volumes.logs.mode=dynamic"}
			_, err := model.HelmTemplate(t, chart, helmArgs, dynamicLogsVolume...)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Volume logs is missing field: dynamic")

			dynamicLogsVolume = append(dynamicLogsVolume, "--set", "volumes.logs.dynamic.storageClassName=neo4j")
			_, err = model.HelmTemplate(t, chart, helmArgs, dynamicLogsVolume...)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "The storage capacity of volumes.logs must be specified")
			assert.Contains(t, err.Error(), "Set volumes.logs.dynamic.requests.storage to a suitable value")

			dynamicLogsVolume = append(dynamicLogsVolume, "--set", "volumes.logs.dynamic.requests.storage=10Gi")
			_, err = model.HelmTemplate(t, chart, helmArgs, dynamicLogsVolume...)
			assert.NoError(t, err)
		}))
}

func TestEnterpriseThrowsErrorIfLicenseAgreementNotAccepted(t *testing.T) {
	t.Parallel()

	testCases := [][]string{
		useEnterprise,
		{"--set", "neo4j.edition=ENTERPRISE"},
		append(useEnterprise, "--set", "neo4j.acceptLicenseAgreement=absolutely"),
		append(useEnterprise, "--set", "neo4j.acceptLicenseAgreement=no"),
		append(useEnterprise, "--set", "neo4j.acceptLicenseAgreement=false"),
		append(useEnterprise, "--set", "neo4j.acceptLicenseAgreement=true"),
		append(useEnterprise, "--set", "neo4j.acceptLicenseAgreement=1"),
		append(useEnterprise, "--set", "neo4j.acceptLicenseAgreement.yes=yes"),
		append(useEnterprise, resources.AcceptLicenseAgreementBoolYes.HelmArgs()...),
		append(useEnterprise, resources.AcceptLicenseAgreementBoolTrue.HelmArgs()...),
	}

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, testCase []string) {
		t.Parallel()
		_, err := model.HelmTemplate(t, chart, testCase)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "to use Neo4j Enterprise Edition you must have a Neo4j license agreement")
		assert.Contains(t, err.Error(), "Set neo4j.acceptLicenseAgreement: \"yes\" to confirm that you have a Neo4j license agreement.")
	}

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChartBuilder) {
		for i, testCase := range testCases {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				doTestCase(t, chart, testCase)
			})
		}
	})
}

func TestEnterpriseDoesNotThrowErrorIfLicenseAgreementAccepted(t *testing.T) {
	t.Parallel()

	testCases := [][]string{
		append(useEnterprise, "--set", "neo4j.acceptLicenseAgreement=yes"),
		append(useEnterprise, acceptLicenseAgreement...),
		append(useEnterprise, resources.AcceptLicenseAgreement.HelmArgs()...),
	}

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, testCase []string) {
		t.Parallel()
		manifest, err := model.HelmTemplate(t, chart, requiredDataMode, testCase...)
		if !assert.NoError(t, err) {
			return
		}

		checkNeo4jManifest(t, manifest)
	}

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChartBuilder) {
		for i, testCase := range testCases {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				doTestCase(t, chart, testCase)
			})
		}
	})
}

// This test is just to check that the produced helm chart doesn't throw any errors
func TestEnterpriseDoesNotThrowIfSet(t *testing.T) {
	t.Parallel()

	testCases := [][]string{
		useEnterpriseAndAcceptLicense,
		//memory is not set here hence it will pick the default value from values.yaml
		append(useEnterpriseAndAcceptLicense, "--set", "neo4j.resources.requests.cpu=500m"),
		//cpu is not set here hence it will pick the default value from values.yaml
		append(useEnterpriseAndAcceptLicense, "--set", "neo4j.resources.requests.memory=2Gi"),
		//cpu is not set here hence it will pick the default value from values.yaml
		append(useEnterpriseAndAcceptLicense, "--set", "neo4j.resources.memory=2Gi"),
		//memory is not set here hence it will pick the default value from values.yaml
		append(useEnterpriseAndAcceptLicense, "--set", "neo4j.resources.cpu=500m"),
		append(useEnterpriseAndAcceptLicense, "--set", "neo4j.resources.cpu=500m", "--set", "neo4j.resources.requests.memory=2Gi"),
		append(useEnterpriseAndAcceptLicense, "--set", "neo4j.resources.requests.cpu=500m", "--set", "neo4j.resources.memory=2Gi"),
		append(useEnterpriseAndAcceptLicense, "--set", "neo4j.resources.requests.cpu=500m", "--set", "neo4j.resources.requests.memory=2Gi"),
		append(useEnterpriseAndAcceptLicense, "--set", "neo4j.resources.cpu=500m", "--set", "neo4j.resources.memory=2Gi"),
		append(useEnterpriseAndAcceptLicense, resources.ApocCorePlugin.HelmArgs()...),
		append(useEnterpriseAndAcceptLicense, resources.CsvMetrics.HelmArgs()...),
		append(useEnterpriseAndAcceptLicense, resources.DefaultStorageClass.HelmArgs()...),
		append(useEnterpriseAndAcceptLicense, resources.JvmAdditionalSettings.HelmArgs()...),
		append(useEnterpriseAndAcceptLicense, resources.PluginsInitContainer.HelmArgs()...),
	}

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, testCase []string) {
		t.Parallel()
		manifest, err := model.HelmTemplate(t, chart, requiredDataMode, testCase...)
		if !assert.NoError(t, err) {
			return
		}

		checkNeo4jManifest(t, manifest)
	}

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChartBuilder) {
		for i, testCase := range testCases {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				doTestCase(t, chart, testCase)
			})
		}
	})
}

// TestEnterpriseContainsDefaultBackupAddress checks if the default backup address is set to 0.0.0.0:6362 or not in enterprise standalone
// and cluster-core charts
func TestEnterpriseContainsDefaultBackupAddress(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChartBuilder) {
		manifest, err := model.HelmTemplate(t, chart, requiredDataMode, useEnterpriseAndAcceptLicense...)
		if !assert.NoError(t, err) {
			return
		}

		configMaps := manifest.OfType(&v1.ConfigMap{})
		for _, configMap := range configMaps {
			cm := configMap.(*v1.ConfigMap)
			if strings.Contains(cm.Name, "default-config") {
				assert.Contains(t, cm.Data["dbms.backup.listen_address"], "0.0.0.0:6362")
			}
		}

	})
}

// Tests the "default" behaviour that you get if you don't pass in *any* other values and the helm chart defaults are used
func TestDefaultEnterpriseHelmTemplate(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChartBuilder) {
		manifest, err := model.HelmTemplate(t, chart, requiredDataMode, useEnterpriseAndAcceptLicense...)
		if !assert.NoError(t, err) {
			return
		}

		checkNeo4jManifest(t, manifest)

		neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		for _, container := range neo4jStatefulSet.Spec.Template.Spec.Containers {
			assert.Contains(t, container.Image, "enterprise")
		}
		for _, container := range neo4jStatefulSet.Spec.Template.Spec.InitContainers {
			assert.Contains(t, container.Image, "enterprise")
		}
	})
}

// TestNeo4jBackupPorts checks for backup ports in the services when backup disabled and enabled
func TestNeo4jBackupPorts(t *testing.T) {
	t.Parallel()

	var helmValues model.HelmValues

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		if edition == "community" {
			helmValues = model.DefaultCommunityValues
			manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues)
			assert.NoError(t, err, "error while testing backup ports when backup is enabled")
			statefulSet := manifest.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
			assert.NotNil(t, statefulSet, fmt.Sprintf("no statefulset found with name %s", model.DefaultHelmTemplateReleaseName))

			ports := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0].Ports
			for _, port := range ports {
				assert.NotEqual(t, port.Name, "backup", "found backup port name in statefulset when backup is disabled")
				assert.NotEqual(t, port.ContainerPort, int32(6362), "found backup port in statefulset when backup is disabled")
			}
		} else {
			helmValues = model.DefaultEnterpriseValues
			manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues)
			assert.NoError(t, err, "error while testing backup ports when backup is enabled")

			services := manifest.OfType(&v1.Service{})
			for _, service := range services {
				svcPorts := service.(*v1.Service).Spec.Ports
				for _, port := range svcPorts {
					if port.Name == "tcp-backup" {
						assert.Equal(t, port.Port, int32(6362), fmt.Sprintf("Backup Port found %d is not matching with 6362", port.Port))
						assert.Equal(t, port.TargetPort.IntValue(), 6362, fmt.Sprintf("Backup Target Port found %d is not matching with 6362", port.TargetPort.IntVal))
					}
				}
			}

			neo4jConfigs := make(map[string]string, 1)
			neo4jConfigs["dbms.backup.enabled"] = "false"
			helmValues.Config = neo4jConfigs
			manifest, err = model.HelmTemplateFromStruct(t, chart, helmValues)
			assert.NoError(t, err, "error while testing backup ports when backup is disabled")

			services = manifest.OfType(&v1.Service{})
			for _, service := range services {
				svcPorts := service.(*v1.Service).Spec.Ports
				for _, port := range svcPorts {
					assert.NotEqual(t, port.Name, "tcp-backup", "found backup port when backup is disabled")
				}
			}

			statefulSet := manifest.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
			assert.NotNil(t, statefulSet, fmt.Sprintf("no statefulset found with name %s", model.DefaultHelmTemplateReleaseName))

			ports := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0].Ports
			for _, port := range ports {
				assert.NotEqual(t, port.Name, "backup", "found backup port name in statefulset when backup is disabled")
				assert.NotEqual(t, port.ContainerPort, int32(6362), "found backup port in statefulset when backup is disabled")
			}
		}

	}))
}

func TestAdditionalEnvVars(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(
		func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
			helmArgs := []string{}
			helmArgs = append(helmArgs, requiredDataMode...)
			if edition == "enterprise" {
				helmArgs = append(helmArgs, acceptLicenseAgreement...)
			}

			manifest, err := model.HelmTemplate(t, chart, helmArgs, "--set", "env.FOO=one", "--set", "env.GRAPHS=are everywhere")
			if !assert.NoError(t, err) {
				return
			}

			envConfigMap := manifest.OfTypeWithName(&v1.ConfigMap{}, model.DefaultHelmTemplateReleaseName.EnvConfigMapName()).(*v1.ConfigMap)
			assert.Equal(t, envConfigMap.Data["FOO"], "one")
			assert.Equal(t, envConfigMap.Data["GRAPHS"], "are everywhere")

			checkNeo4jManifest(t, manifest)
		}))
}

func TestBoolsInConfig(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, s string) {
		_, err := model.HelmTemplateFromYamlFile(t, chart, resources.BoolsInConfig, acceptLicenseAgreement...)
		assert.Error(t, err, "Helm chart should fail if config contains boolean values")
		assert.Contains(t, err.Error(), "config values must be strings.")
		assert.Contains(t, err.Error(), "metrics.enabled")
		assert.Contains(t, err.Error(), "type: bool")
	}))
}

func TestIntsInConfig(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, s string) {
		_, err := model.HelmTemplateFromYamlFile(t, chart, resources.IntsInConfig, acceptLicenseAgreement...)
		assert.Error(t, err, "Helm chart should fail if config contains int values")
		assert.Contains(t, err.Error(), "config values must be strings.")
		assert.Contains(t, err.Error(), "metrics.csv.rotation.keep_number")
		assert.Contains(t, err.Error(), "type: float64")
	}))
}

// Tests the "default" behaviour that you get if you don't pass in *any* other values and the helm chart defaults are used
func TestChmodInitContainer(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, s string) {
		manifest, err := model.HelmTemplateFromYamlFile(t, chart, resources.ChmodInitContainer, acceptLicenseAgreement...)
		if !assert.NoError(t, err) {
			return
		}

		checkNeo4jManifest(t, manifest)

		neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		initContainers := neo4jStatefulSet.Spec.Template.Spec.InitContainers
		assert.Len(t, initContainers, 1)
		container := initContainers[0]
		assert.Equal(t, "set-volume-permissions", container.Name)
		assert.Len(t, container.VolumeMounts, 6)
		// Command will chown logs
		assert.Contains(t, container.Command[2], "chown -R \"7474\" \"/logs\"")
		assert.Contains(t, container.Command[2], "chgrp -R \"7474\" \"/logs\"")
		assert.Contains(t, container.Command[2], "chmod -R g+rwx \"/logs\"")
		// Command will not chown data
		assert.NotContains(t, container.Command[2], "chown -R \"7474\" \"/data\"")
		assert.NotContains(t, container.Command[2], "chgrp -R \"7474\" \"/data\"")
		assert.NotContains(t, container.Command[2], "chmod -R g+rwx \"/data\"")
	}))
}

// Tests the "default" behaviour that you get if you don't pass in *any* other values and the helm chart defaults are used
func TestChmodInitContainers(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, s string) {
		manifest, err := model.HelmTemplateFromYamlFile(t, chart, resources.ChmodInitContainerAndCustomInitContainer, acceptLicenseAgreement...)
		if !assert.NoError(t, err) {
			return
		}

		checkNeo4jManifest(t, manifest)

		neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		initContainers := neo4jStatefulSet.Spec.Template.Spec.InitContainers
		assert.Len(t, initContainers, 2)
		container := initContainers[0]
		assert.Equal(t, "set-volume-permissions", container.Name)
		assert.Len(t, container.VolumeMounts, 6)
		// Command will chown logs
		assert.Contains(t, container.Command[2], "chown -R \"7474\" \"/logs\"")
		assert.Contains(t, container.Command[2], "chgrp -R \"7474\" \"/logs\"")
		assert.Contains(t, container.Command[2], "chmod -R g+rwx \"/logs\"")
		// Command will not chown data
		assert.NotContains(t, container.Command[2], "chown -R \"7474\" \"/data\"")
		assert.NotContains(t, container.Command[2], "chgrp -R \"7474\" \"/data\"")
		assert.NotContains(t, container.Command[2], "chmod -R g+rwx \"/data\"")
	}))
}

type authSecretTest struct {
	neo4jName      *string
	setPassword    bool
	password       *string
	expectedResult authSecretExpectation
}

type authSecretExpectation struct {
	helmFailsWithError     error
	authSecretCreated      bool
	randomPasswordAssigned bool
}

func (a authSecretTest) PasswordFlag() string {
	if a.setPassword == true {
		return `true`
	}
	return `false`
}

func (a authSecretTest) String() (str string) {
	str = fmt.Sprintf("setPassword:%v;password:", a.setPassword)
	if a.password == nil {
		return str + "nil"
	}

	return str + *a.password
}

func getNeo4jPassword(authSecret *v1.Secret) string {
	b64Value := authSecret.Data["NEO4J_AUTH"]
	return string(b64Value)
}

var emptyString = ""

func TestAuthSecrets(t *testing.T) {
	t.Parallel()

	var neo4jDotName = "secret-test"
	testCases := []authSecretTest{
		{&neo4jDotName, false, nil, authSecretExpectation{authSecretCreated: false}},
		{nil, false, nil, authSecretExpectation{authSecretCreated: false}},
		{&neo4jDotName, false, &emptyString, authSecretExpectation{authSecretCreated: false}},
		{nil, false, &emptyString, authSecretExpectation{authSecretCreated: false}},
		{&neo4jDotName, true, &model.DefaultPassword, authSecretExpectation{authSecretCreated: true}},
		{nil, true, &model.DefaultPassword, authSecretExpectation{authSecretCreated: true}},
		{&neo4jDotName, true, nil, authSecretExpectation{authSecretCreated: true, randomPasswordAssigned: true}},
		{nil, true, nil, authSecretExpectation{authSecretCreated: true, randomPasswordAssigned: true}},
		{&neo4jDotName, true, &emptyString, authSecretExpectation{authSecretCreated: true, randomPasswordAssigned: true}},
		{nil, true, &emptyString, authSecretExpectation{authSecretCreated: true, randomPasswordAssigned: true}},
		{&neo4jDotName, false, &model.DefaultPassword, authSecretExpectation{helmFailsWithError: errors.New("unsupported State: Cannot set neo4j.password or neo4j.passwordFromSecret when Neo4j authis disabled (dbms.security.auth_enabled=false). Either remove neo4j.password setting or enable Neo4j auth")}},
		{nil, false, &model.DefaultPassword, authSecretExpectation{helmFailsWithError: errors.New("unsupported State: Cannot set neo4j.password or neo4j.passwordFromSecret when Neo4j authis disabled (dbms.security.auth_enabled=false). Either remove neo4j.password setting or enable Neo4j auth")}},
	}

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string, testCase authSecretTest) {
		t.Parallel()
		expectation := testCase.expectedResult

		helmArgs := []string{
			"--set", "neo4j.edition=" + edition,
			"--set-string", `config.dbms\.security\.auth_enabled=` + testCase.PasswordFlag(),
		}

		if testCase.neo4jName != nil {
			helmArgs = append(helmArgs, "--set", "neo4j.name="+*testCase.neo4jName)
		}

		if testCase.password != nil {
			helmArgs = append(helmArgs, "--set", "neo4j.password="+*testCase.password)
		}

		if edition == "enterprise" {
			helmArgs = append(helmArgs, "--set", "neo4j.acceptLicenseAgreement=yes")
		}

		manifest, err := model.HelmTemplate(t, chart, requiredDataMode, helmArgs...)

		if expectation.helmFailsWithError != nil {
			assert.Contains(t, err.Error(), expectation.helmFailsWithError.Error())
			return
		}

		if !assert.NoError(t, err) {
			return
		}

		secrets := manifest.OfType(&v1.Secret{})

		if expectation.authSecretCreated {
			assert.Len(t, secrets, 1)
			authSecret := secrets[0].(*v1.Secret)

			// Slightly complicated set of rules here. The reason is neo4j-cluster charts default neo4j.name to 'neo4j-cluster' but the neo4j-standalone chart
			// defaults the neo4j.name to the name of the release.
			if testCase.neo4jName != nil {
				assert.Equal(t, *testCase.neo4jName+"-auth", authSecret.Name)
			} else if chart.Name() == "neo4j-standalone" {
				assert.Equal(t, "my-release-auth", authSecret.Name)
			} else {
				assert.Equal(t, "neo4j-cluster-auth", authSecret.Name)
			}

			password := getNeo4jPassword(authSecret)
			defaultHelmPasswordPrefix := "neo4j/defaulthelmpassword"
			if expectation.randomPasswordAssigned {
				assert.Equal(t, "neo4j/", password[0:6])
				assert.Greater(t, len(password), len("neo4j/123"))
				assert.NotContains(t, password, defaultHelmPasswordPrefix)
			} else {
				assert.Equal(t, "neo4j/"+*testCase.password, password)
				assert.Contains(t, password, defaultHelmPasswordPrefix)
			}

		} else {
			assert.Len(t, secrets, 0)
		}

		checkNeo4jManifest(t, manifest)
	}

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		for i, testCase := range testCases {
			t.Run(fmt.Sprintf("%d %s", i, testCase), func(t *testing.T) {
				doTestCase(t, chart, edition, testCase)
			})
		}
	}))
}

func TestPasswordFromExistingSecretWithLookupDisabled(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.Neo4J.PasswordFromSecret = "test-secret"
		helmValues.DisableLookups = true

		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues)
		if !assert.NoError(t, err) {
			return
		}
		statefulSet := manifest.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
		if !assert.NotNil(t, statefulSet, fmt.Sprintf("no statefulset found with name %s", model.DefaultHelmTemplateReleaseName)) {
			return
		}
		neo4jContainer := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0]
		assert.NotContains(t, neo4jContainer.EnvFrom, v1.EnvFromSource{
			SecretRef: &v1.SecretEnvSource{
				LocalObjectReference: v1.LocalObjectReference{Name: "test-secret"},
			},
		})
	}))
}

func TestDefaultLabels(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		neo4jName := strings.ToLower(t.Name())
		expectedLabels := map[string]string{
			"app": neo4jName,
		}

		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense,
			"--set", "neo4j.name="+neo4jName)
		if !assert.NoError(t, err) {
			return
		}

		metadata := manifest.AllWithMetadata()
		if chart == model.ClusterCoreHelmChart {
			assert.Len(t, metadata, 12)
		} else {
			assert.Len(t, metadata, 9)
		}

		for _, object := range metadata {
			actualLabels := object.GetLabels()
			for key, expectedValue := range expectedLabels {
				assert.Contains(t, actualLabels, key, fmt.Sprintf("K8s %s object '%s' is missing expected label %s", object.(runtime.Object).GetObjectKind(), object.GetName(), key))
				actualValue := actualLabels[key]
				assert.Equal(t, expectedValue, actualValue, fmt.Sprintf("K8s %s object '%s' has unexpected value for label '%s'. expected: %s; actual: %s;", object.(runtime.Object).GetObjectKind(), object.GetName(), key, expectedValue, actualValue))
			}
		}
	}))
}

// TestNeo4jPodAnnotations checks if Neo4j Pod has the annotations or not
func TestNeo4jPodAnnotations(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.PodSpecAnnotations.HelmArgs()...)
		assert.NoError(t, err)

		statefulSet := manifest.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
		assert.NotNil(t, statefulSet, fmt.Sprintf("no statefulset found with name %s", model.DefaultHelmTemplateReleaseName))

		podAnnotations := statefulSet.(*appsv1.StatefulSet).Spec.Template.Annotations
		assert.NotNil(t, podAnnotations, "no pod annotations found")

		assert.Contains(t, podAnnotations, "demoKey", "missing podAnnotation demoKey")

		assert.Equal(t, podAnnotations["demoKey"], "alpha", "invalid podAnnotation value for key=demoKey")

		assert.Contains(t, podAnnotations, fmt.Sprintf("checksum/%s-env", model.DefaultHelmTemplateReleaseName.String()), "missing env checksum annotation")

	}))
}

// TestNeo4jStatefulSetAnnotations checks if Neo4j StatefulSet has the annotations or not
func TestNeo4jStatefulSetAnnotations(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.StatefulSetAnnotations.HelmArgs()...)
		assert.NoError(t, err)

		statefulSet := manifest.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
		assert.NotNil(t, statefulSet, fmt.Sprintf("no statefulset found with name %s", model.DefaultHelmTemplateReleaseName))

		statefulSetAnnotations := statefulSet.(*appsv1.StatefulSet).ObjectMeta.Annotations
		assert.NotNil(t, statefulSetAnnotations, "no statefulset annotations found")

		assert.Contains(t, statefulSetAnnotations, "demoKey", "missing statefulSet annotation demoKey")

		assert.Equal(t, statefulSetAnnotations["demoKey"], "alpha", "invalid statefulSet annotation value for key=demoKey")

	}))
}

// TestNeo4jPodPriorityClassName checks for Neo4j PriorityClassName
// error should be thrown since we are not creating priorityClass in the cluster in advance
func TestNeo4jPodPriorityClassName(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		_, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.PriorityClassName.HelmArgs()...)
		if !assert.Error(t, err, "error should be thrown for priorityClass") {
			return
		}

		if assert.Contains(t, err.Error(), "PriorityClass demo is missing in the cluster", fmt.Sprintf("priorityClass error message missing \n err := %s", err.Error())) {
			return
		}

	}))
}

// TestNeo4jPodPriorityClassNameWithLookupDisabled checks for Neo4j PriorityClassName with disableLookups flag set to true
func TestNeo4jPodPriorityClassNameWithLookupDisabled(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.PodSpec.PriorityClassName = "demo"
		_, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}
	}))
}

// TestNeo4jPodTolerations checks for tolerations in the statefulset
func TestNeo4jPodTolerations(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.Tolerations.HelmArgs()...)
		if !assert.NoError(t, err) {
			return
		}
		neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		podTolerations := neo4jStatefulSet.Spec.Template.Spec.Tolerations
		if !assert.Len(t, podTolerations, 2, "more than two tolerations found") {
			return
		}
		var invalidTolerationFound bool
		for _, podToleration := range podTolerations {
			if podToleration.Key != "key1" && podToleration.Key != "key2" {
				invalidTolerationFound = true
				break
			}
		}

		if !assert.NotEqual(t, invalidTolerationFound, true, "invalid toleration found") {
			return
		}

	}))
}

// TestNeo4jPodNodeAffinity checks for nodeAffinity setting in statefulset
func TestNeo4jPodNodeAffinity(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.NodeAffinity.HelmArgs()...)
		if !assert.NoError(t, err) {
			return
		}
		neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		nodeAffinity := neo4jStatefulSet.Spec.Template.Spec.Affinity.NodeAffinity
		if !assert.NotNil(t, nodeAffinity, "nil nodeAffinity found") {
			return
		}
		if !assert.NotNil(t, nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution) {
			return
		}
		if !assert.NotEqual(t, len(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution), 0) {
			return
		}
	}))
}

// TestNeo4jPodAntiAffinity checks for podAntiAffinity setting in statefulset
func TestNeo4jPodAntiAffinity(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultEnterpriseValues

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		if edition == "community" {
			helmValues = model.DefaultCommunityValues
		}
		helmValues.PodSpec.PodAntiAffinity = model.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []model.RequiredDuringSchedulingIgnoredDuringExecution{
				{
					LabelSelector: model.LabelSelector{
						MatchLabels: map[string]string{
							"demo": "demo",
						},
					},
					TopologyKey: "demo",
				},
			},
		}
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues)
		assert.NoError(t, err)
		neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		podAntiAffinity := neo4jStatefulSet.Spec.Template.Spec.Affinity.PodAntiAffinity
		assert.NotNil(t, podAntiAffinity, "nil podAntiAffinity found")
		assert.NotNil(t, podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		assert.NotEqual(t, len(podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution), 0)

		helmValues.PodSpec.PodAntiAffinity = false
		manifest, err = model.HelmTemplateFromStruct(t, chart, helmValues)
		neo4jStatefulSet = manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		affinity := neo4jStatefulSet.Spec.Template.Spec.Affinity
		assert.Nil(t, affinity, "affinity found")

		helmValues.PodSpec.PodAntiAffinity = true
		manifest, err = model.HelmTemplateFromStruct(t, chart, helmValues)
		assert.NoError(t, err)
		neo4jStatefulSet = manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		podAntiAffinity = neo4jStatefulSet.Spec.Template.Spec.Affinity.PodAntiAffinity
		assert.NotNil(t, podAntiAffinity, "nil podAntiAffinity found")
		assert.NotNil(t, podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		assert.NotEqual(t, len(podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution), 0)
	}))
}

func TestExtraLabels(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		expectedValue := strconv.Itoa(helpers.RandomIntBetween(0, 1000))
		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense,
			"--set-string", fmt.Sprintf("neo4j.labels.testlabel=%s", expectedValue))
		if !assert.NoError(t, err) {
			return
		}

		testLabel := "testlabel"
		for _, object := range manifest.AllWithMetadata() {
			actualLabels := object.GetLabels()
			assert.Contains(t, actualLabels, testLabel, fmt.Sprintf("K8s %s object '%s' is missing expected label %s", object.(runtime.Object).GetObjectKind(), object.GetName(), testLabel))
			actualValue := actualLabels[testLabel]
			assert.Equal(t, expectedValue, actualValue, fmt.Sprintf("K8s %s object '%s' has unexpected value for label '%s'. expected: %s; actual: %s;", object.(runtime.Object).GetObjectKind(), object.GetName(), testLabel, expectedValue, actualValue))
		}
	}))
}

func TestEmptyImageCredentials(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		_, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.EmptyImageCredentials.HelmArgs()...)
		if !assert.Error(t, err) {
			return
		}
		if !assert.Contains(t, err.Error(), "password field cannot be empty") {
			return
		}
		if !assert.Contains(t, err.Error(), "name field cannot be empty") {
			return
		}
		if !assert.Contains(t, err.Error(), "username field cannot be empty for imageCredential,email field cannot be empty for imageCredential") {
			return
		}
	}))
}

func TestDuplicateImageCredentials(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		_, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.DuplicateImageCredentials.HelmArgs()...)
		if !assert.Error(t, err) {
			return
		}
		if !assert.Contains(t, err.Error(), "Duplicate \"names\" found in imageCredentials list. Please remove duplicates") {
			return
		}
	}))
}

// TestMissingApocConfigMap ensure the apoc config map is not created when apoc configs are not provided in values.yaml
func TestMissingApocConfigMap(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense)
		if !assert.NoError(t, err) {
			return
		}
		configMaps := manifest.OfType(&v1.ConfigMap{})
		if !assert.NotEqual(t, len(configMaps), 0) {
			return
		}
		var apocConfigMapPresent bool
		for _, configMap := range configMaps {
			if strings.Contains(configMap.(*v1.ConfigMap).Name, "apoc-config") {
				apocConfigMapPresent = true
				break
			}
		}
		//apoc-config configMap should not be present since we are not passing apoc configs
		if !assert.Equal(t, apocConfigMapPresent, false, "apocConfigMap present !!! Should NOT BE present infact") {
			return
		}
	}))
}

// TestApocConfigMapIsPresent ensure the apoc config map is present when apoc_config is provided in the values.yaml
func TestApocConfigMapIsPresent(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.ApocConfig.HelmArgs()...)
		if !assert.NoError(t, err) {
			return
		}
		configMaps := manifest.OfType(&v1.ConfigMap{})
		if !assert.NotEqual(t, len(configMaps), 0, "Number of configMaps are zero !!") {
			return
		}
		var apocConfigMapPresent bool
		var apocConfigMap *v1.ConfigMap
		for _, configMap := range configMaps {
			if strings.Contains(configMap.(*v1.ConfigMap).Name, "apoc-config") {
				apocConfigMapPresent = true
				apocConfigMap = configMap.(*v1.ConfigMap)
				break
			}
		}
		//apoc-config configMap should not be present since we are not passing apoc configs
		if !assert.Equal(t, apocConfigMapPresent, true, "apocConfigMap not present !!! Should be present infact") {
			return
		}
		apocConfigYamlDataMap, err := resources.ApocConfig.Data()
		if !assert.NoError(t, err, fmt.Sprintf("error while reading apocConfig.yaml file \n %s", err)) {
			return
		}
		value, present := apocConfigMap.Data["apoc.conf"]
		if !assert.Equal(t, present, true, "configMap does not contain apoc.conf as key !!") {
			return
		}
		apocConfigMapData := strings.Split(value, "\n")
		var yamlData []string
		for key, _ := range apocConfigYamlDataMap {
			valueMap := apocConfigYamlDataMap[key].(map[string]interface{})
			for k, _ := range valueMap {
				yamlData = append(yamlData, fmt.Sprintf("%s=%s", k, valueMap[k]))
			}
		}
		sort.Strings(yamlData)
		sort.Strings(apocConfigMapData)
		if !assert.Equal(t, yamlData, apocConfigMapData,
			fmt.Sprintf("yaml and configMapData mismatch \n "+
				"yamlData %s \n "+
				"configMapData %s",
				yamlData, apocConfigMapData)) {
			return
		}
	}))
}

// TestApocCredentialsArePresentInConfigMap ensure the apoc config map contains the apoc credentials reference
func TestApocCredentialsArePresentInConfigMap(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.Volumes.Data.Mode = "defaultStorageClass"

		helmValues.ApocCredentials = model.ApocCredentials{
			Jdbc: map[string]string{
				"aliasName":       "jdbc",
				"secretName":      "jdbcsecret",
				"secretMountPath": "/secret/jdbcCred",
			},
		}
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoErrorf(t, err, fmt.Sprintf("error while trying to perform helm template with apoc credentials \n %v", err)) {
			return
		}

		configMaps := manifest.OfType(&v1.ConfigMap{})
		assert.NotEqual(t, len(configMaps), 0, "Number of configMaps are zero !!")

		var apocConfigMapPresent bool
		var apocConfigMap *v1.ConfigMap
		for _, configMap := range configMaps {
			if strings.Contains(configMap.(*v1.ConfigMap).Name, "apoc-config") {
				apocConfigMapPresent = true
				apocConfigMap = configMap.(*v1.ConfigMap)
				break
			}
		}
		//apoc-config configMap should not be present since we are not passing apoc configs
		assert.Equal(t, apocConfigMapPresent, true, "apocConfigMap not present !!! Should be present")

		value, present := apocConfigMap.Data["apoc.conf"]
		assert.Equal(t, present, true, "configMap does not contain apoc.conf as key !!")

		assert.Contains(t, value, "apoc.jdbc.jdbc.url=$(bash -c 'cat /secret/jdbcCred/URL')")

	}))
}

// TestApocCredentialsVolumesAndMounts ensure that the required volume mounts and volumes are present or not when apoc_credentials are specified
func TestApocCredentialsVolumesAndMounts(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true

		helmValues.ApocCredentials = model.ApocCredentials{
			Jdbc: map[string]string{
				"aliasName":       "jdbc",
				"secretName":      "jdbcsecret",
				"secretMountPath": "/secret/jdbcCred",
			},
		}
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		statefulSet := manifest.OfType(&appsv1.StatefulSet{})[0]
		assert.NotNil(t, statefulSet, "statefulset missing")

		volumes := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Volumes
		assert.NotEqual(t, len(volumes), 0, "no volumes found")
		var names []string
		for _, volume := range volumes {
			names = append(names, volume.Name)
		}
		assert.Contains(t, names, "apoc-jdbc-url", "missing apoc credentials jdbc volume")

		volumeMounts := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0].VolumeMounts
		assert.NotEqual(t, len(volumeMounts), 0, "no volume mounts found")
		names = []string{}
		var mountPath []string
		for _, volumeMount := range volumeMounts {
			names = append(names, volumeMount.Name)
			mountPath = append(mountPath, volumeMount.MountPath)
		}
		assert.Contains(t, names, "apoc-jdbc-url", "missing apoc credentials jdbc volume mount")
		assert.Contains(t, mountPath, "/secret/jdbcCred", "missing apoc credentials jdbc volume mount path")
	}))
}

func TestMissingImageCredentials(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		_, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.MissingImageCredentials.HelmArgs()...)
		if !assert.Error(t, err) {
			return
		}
		if !assert.Contains(t, err.Error(), "Missing imageCredential entry") {
			return
		}
	}))
}

// TestEmptyImagePullSecrets ensures empty imagePullSecret names or names with just spaces are not included in the cluster formation
func TestEmptyImagePullSecrets(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.EmptyImagePullSecrets.HelmArgs()...)
		if !assert.NoError(t, err) {
			return
		}
		secrets := manifest.OfType(&v1.Secret{})
		secret1 := manifest.OfTypeWithName(&v1.Secret{}, "secret1").(*v1.Secret)
		secret2 := manifest.OfTypeWithName(&v1.Secret{}, "secret2").(*v1.Secret)

		if !assert.NotEqual(t, len(secrets), 0) {
			fmt.Errorf("No secrets found !!")
			return
		}
		var secretCount int
		for _, secret := range secrets {
			if secret.(*v1.Secret).Type == "kubernetes.io/dockerconfigjson" {
				secretCount++
			}
		}
		if !assert.Equal(t, secretCount, 2) {
			fmt.Errorf("%d secrets of type \"kubernetes.io/dockerconfigjson\" found instead of 2 ", secretCount)
			return
		}
		if !assert.NotNil(t, secret1) {
			return
		}
		if !assert.NotNil(t, secret2) {
			return
		}
		if !assert.Equal(t, secret1.Name, "secret1") {
			fmt.Errorf(" secret name %s not matching with secret1", secret1.Name)
			return
		}
		if !assert.Equal(t, secret2.Name, "secret2") {
			fmt.Errorf(" secret name %s not matching with secret2", secret2.Name)
			return
		}
	}))
}

func TestInvalidNodeSelectorLabels(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		_, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.InvalidNodeSelectorLabels.HelmArgs()...)
		if !assert.Error(t, err) {
			return
		}
		if !assert.Contains(t, err.Error(), "No node exists in the cluster which has all the below labels") {
			t.Logf("Invalid nodeselector error message")
			return
		}
	}))
}

// TestNodeSelectorLabelsWithLookupDisabledWithDryRun tests disableLookups flag when set to false along with --dry-run flag and nodeSelector labels
func TestNodeSelectorLabelsWithLookupDisabledWithDryRun(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.NodeSelector = map[string]string{
			"label1": "value1",
		}
		_, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}
	}))
}

// TestImagePullSecretWithLookupsDisabledWithDryRun tests disableLookups flag when set to true along with --dry-run flag and imagePullSecrets
func TestImagePullSecretWithLookupsDisabledWithDryRun(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		imagePullSecretName := "demo"
		helmValues.Image.ImagePullSecrets = []string{imagePullSecretName}
		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, "error seen while testing helm chart with imagePullSecrets with Disabled Lookups")
		secrets := manifests.OfType(&v1.Secret{})
		for _, secret := range secrets {
			secretName := secret.(*v1.Secret).Name
			assert.NotEqual(t, secretName, imagePullSecretName, fmt.Sprintf("Found a secret %s matching with imagePullSecret %s", secretName, imagePullSecretName))
		}
	}))
}

// TestContainerSecurityContext tests the presence of default containerSecurityContext in statefulSet
func TestContainerSecurityContext(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}

		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues)
		if !assert.NoError(t, err) {
			return
		}

		statefulSet := manifest.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
		if !assert.NotNil(t, statefulSet, "statefulset found to be empty") {
			return
		}
		containerSecurityContext := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0].SecurityContext
		if !assert.Equal(t, *containerSecurityContext.RunAsNonRoot, true, fmt.Sprintf("runAsNonRoot current value %s does not match with %s", strconv.FormatBool(*containerSecurityContext.RunAsNonRoot), "true")) {
			return
		}
		if !assert.Equal(t, *containerSecurityContext.RunAsUser, int64(7474), fmt.Sprintf("runAsUser current value %d does not match with %s", *containerSecurityContext.RunAsUser, "7474")) {
			return
		}
		if !assert.Equal(t, *containerSecurityContext.RunAsGroup, int64(7474), fmt.Sprintf("runAsGroup current value %d does not match with %s", *containerSecurityContext.RunAsGroup, "7474")) {
			return
		}
	}))
}

// TestAdditionalVolumesAndMounts checks if the additionalVolumes and additionalVolumeMounts are present or not
func TestAdditionalVolumesAndMounts(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.AdditionalVolumesAndMounts.HelmArgs()...)
		if !assert.NoError(t, err) {
			return
		}
		neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		if !assert.NotNil(t, neo4jStatefulSet, "missing statefulSet object") {
			return
		}
		podVolumes := neo4jStatefulSet.Spec.Template.Spec.Volumes
		if !assert.Len(t, podVolumes, 3, fmt.Sprintf("more than three volumes present")) {
			return
		}
		var volumePresent bool
		for _, podVolumes := range podVolumes {
			if podVolumes.Name == "neo4j1-conf" {
				volumePresent = true
				break
			}
		}
		if !assert.True(t, volumePresent, "neo4j1-conf volume not found") {
			return
		}

		containers := neo4jStatefulSet.Spec.Template.Spec.Containers
		if !assert.Len(t, containers, 1, "more than one container found") {
			return
		}
		volumeMounts := containers[0].VolumeMounts
		var volumeMountPresent bool
		for _, volumeMount := range volumeMounts {
			if volumeMount.Name == "neo4j1-conf" {
				if volumeMount.MountPath == "/config/neo4j1.conf" {
					volumeMountPresent = true
					break
				}
			}
		}
		if !assert.True(t, volumeMountPresent, "neo4j1-conf volumeMount not found") {
			return
		}

	}))
}

// TestVolumeClaimLabels checks if provided labels are getting attached to the pvc metadata or not
func TestVolumeClaimLabels(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		labels := make(map[string]string, 2)
		labels["key1"] = "value1"
		labels["key2"] = "value2"
		helmValues.Volumes.Data.Labels = labels
		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, "no error should be seen while checking for labels on volume claims")
		statefulSet := manifests.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
		if !assert.NotNil(t, statefulSet, fmt.Sprintf("no statefulset found with name %s", model.DefaultHelmTemplateReleaseName)) {
			return
		}

		volumeClaims := statefulSet.(*appsv1.StatefulSet).Spec.VolumeClaimTemplates
		assert.Equal(t, len(volumeClaims), 1, fmt.Sprintf("only one volume claim should be present ,found %d", len(volumeClaims)))
		assert.Equal(t, volumeClaims[0].ObjectMeta.Labels, labels, "labels are not matching")
	}))
}

// TestNeo4jConfigWithEmptyLdapPasswordFromSecretAndMountPath checks that the user-config configmap does not contains ldapPassword config when empty ldapPasswordFromSecret and ldapPasswordMountPath are used
func TestNeo4jConfigWithEmptyLdapPasswordFromSecretAndMountPath(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.LdapPasswordFromSecret = "  "
		helmValues.LdapPasswordMountPath = "  "
		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}
		configMap := manifests.OfTypeWithName(&v1.ConfigMap{}, fmt.Sprintf("%s-user-config", model.DefaultHelmTemplateReleaseName.String()))
		data := configMap.(*v1.ConfigMap).Data
		assert.NotContains(t, data, "dbms.security.ldap.authorization.system_password", "Ldap Password Config should not be present")
	}))
}

// TestNeo4jConfigWithEmptyLdapPasswordFromSecret checks for error when empty ldapPasswordFromSecret is provided along with a mount path
func TestNeo4jConfigWithEmptyLdapPasswordFromSecret(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.LdapPasswordFromSecret = "  "
		helmValues.LdapPasswordMountPath = "/config/ldapPassword"
		_, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if edition == "enterprise" {
			assert.Error(t, err, "error should be seen when passing empty ldapPasswordFromSecret along with valid mount path")
			assert.Contains(t, err.Error(), "Please define 'ldapPasswordFromSecret'")
		} else {
			assert.Error(t, err, "error should be seen when using ldapPasswordMount with community edition")
			assert.Contains(t, err.Error(), "ldapPasswordFromSecret and ldapPasswordMountPath are Enterprise Edition feature only")
		}

	}))
}

// TestNeo4jConfigWithEmptyLdapPasswordMountPath checks for error when empty ldapPasswordMountPath is provided along with a valid ldapPasswordFromSecret
func TestNeo4jConfigWithEmptyLdapPasswordMountPath(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.LdapPasswordFromSecret = "ldapsecret"
		helmValues.LdapPasswordMountPath = "  "
		_, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if edition == "enterprise" {
			assert.Error(t, err, "error should be seen when passing empty ldapPasswordMountPath along with valid ldap secret")
			assert.Contains(t, err.Error(), "Please define 'ldapPasswordMountPath'")
		} else {
			assert.Error(t, err, "error should be seen when using ldapPasswordFromSecret with community edition")
			assert.Contains(t, err.Error(), "ldapPasswordFromSecret and ldapPasswordMountPath are Enterprise Edition feature only")
		}
	}))
}

// TestLdapVolumeAndVolumeMountsExistsOrNot checks for ldap volume and volume-mount when ldapPasswordFromSecret and ldapPasswordMountPath are provided
func TestLdapVolumeAndVolumeMountsExistsOrNot(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.LdapPasswordFromSecret = "ldapsecret"
		helmValues.LdapPasswordMountPath = "/config/ldapPassword"
		helmValues.DisableLookups = true
		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if edition == "enterprise" {
			assert.NoError(t, err, "no error should be seen while checking for ldap volume and volumeMount")
			statefulSet := manifests.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
			if !assert.NotNil(t, statefulSet, fmt.Sprintf("no statefulset found with name %s", model.DefaultHelmTemplateReleaseName)) {
				return
			}

			volumes := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Volumes
			assert.NotEqual(t, len(volumes), 0, "no volumes found")
			var names []string
			for _, volume := range volumes {
				names = append(names, volume.Name)
			}
			assert.Contains(t, names, "neo4j-ldap-password", "missing neo4j-ldap-password volume")

			volumeMounts := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0].VolumeMounts
			assert.NotEqual(t, len(volumeMounts), 0, "no volume mounts found")
			names = []string{}
			for _, volumeMount := range volumeMounts {
				names = append(names, volumeMount.Name)
			}
			assert.Contains(t, names, "neo4j-ldap-password", "missing neo4j-ldap-password volume mount")
		} else {
			assert.Error(t, err, "error should be seen when using ldapPasswordFromSecret with community edition")
			assert.Contains(t, err.Error(), "ldapPasswordFromSecret and ldapPasswordMountPath are Enterprise Edition feature only")
		}
	}))
}

func TestErrorIsThrownForInvalidMemoryResources(t *testing.T) {

	t.Parallel()
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		//valid syntax but less than the minimum 2G or 2Gi
		invalidMemory := []string{
			"2M",
			"1000K",
			"12345.67",
			"0.1234",
			"123e+5",
			"0.0003T",
		}
		checkMemoryResources(t, chart, edition, invalidMemory, "less than minimum")
	}
	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestErrorIsThrownForInvalidMemoryResourcesRegex(t *testing.T) {

	t.Parallel()
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		invalidMemoryRegexs := []string{
			"P1233",
			"2.G",
			"1.e",
			"1.23ei",
			"2.5GG",
			"2.3i",
			"1.eeee",
			"1.2.3.3.4",
			"1. ",
			"1.34ki",
			"1.i.1.2",
			"1.iK",
			"1.3456I",
			"2.5 K ",
			"123.B",
		}
		checkMemoryResources(t, chart, edition, invalidMemoryRegexs, "Invalid memory value")
	}
	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestErrorIsThrownForEmptyMemoryResources(t *testing.T) {

	t.Parallel()
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		invalidMemory := []string{
			"",
		}
		checkMemoryResources(t, chart, edition, invalidMemory, "neo4j.resources.memory cannot be set to \"\"")
	}
	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestErrorIsThrownForEmptyCPUResources(t *testing.T) {

	t.Parallel()
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		invalidCPU := []string{
			"",
		}
		checkCPUResources(t, chart, edition, invalidCPU, "neo4j.resources.cpu cannot be set to \"\"")
	}
	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestErrorIsThrownForInvalidCPUResourcesRegex(t *testing.T) {

	t.Parallel()
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		invalidCPURegexs := []string{
			"123m123m123",
			"123m3222",
			"1.2.3.4.4.",
			"m12334",
			"1.2.3.4",
			"m",
			"1.m",
			"1.A",
			"1.2.3.m.4.4.",
			"1.m.3.4.m",
			"1m2m3m3m",
			"m1233",
			"1..23442",
		}
		checkCPUResources(t, chart, edition, invalidCPURegexs, "Invalid cpu value")
	}
	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestErrorIsThrownForInvalidCPUResources(t *testing.T) {

	t.Parallel()
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		//valid syntax but less than the minimum 2G or 2Gi
		invalidCPU := []string{
			"1.5m",
			"123.45m",
			"0.5m",
			"0.12343",
		}
		checkCPUResources(t, chart, edition, invalidCPU, "less than minimum")
	}
	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestNeo4jResourcesAndLimits(t *testing.T) {

	t.Parallel()

	var testCases = []Neo4jResourceTestCase{
		GenerateNeo4jResourcesTestCase([]string{"cpuResources"}, "500m", ""),
		GenerateNeo4jResourcesTestCase([]string{"cpuRequests"}, "1", ""),
		GenerateNeo4jResourcesTestCase([]string{"memoryResources"}, "", "3Gi"),
		GenerateNeo4jResourcesTestCase([]string{"memoryRequests"}, "", "3Gi"),
		GenerateNeo4jResourcesTestCase([]string{"cpuRequests", "memoryResources"}, "1", "3Gi"),
		GenerateNeo4jResourcesTestCase([]string{"cpuResources", "memoryResources"}, "0.5", "3Gi"),
		GenerateNeo4jResourcesTestCase([]string{"cpuRequests", "memoryRequests"}, "0.5", "3Gi"),
		GenerateNeo4jResourcesTestCase([]string{"cpuResources", "memoryRequests"}, "0.5", "3Gi"),
	}

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		for i, testCase := range testCases {
			t.Run(fmt.Sprintf("%d %s", i, testCase), func(t *testing.T) {
				checkResourcesAndLimits(t, chart, edition, testCase)
			})
		}
	}))
}

// TestDisableSubPathExprFlag tests disableSubPathExpr flag when set to true along with --dry-run flag
func TestDisableSubPathExprFlag(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.Volumes.Licenses.DisableSubPath = true

		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, fmt.Sprintf("error while testing disableSubPathExpr \n%v", err))

		neo4jStatefulSet := manifests.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		assert.NotNil(t, neo4jStatefulSet, "missing statefulSet object")

		volumeMounts := neo4jStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts
		assert.NotEqual(t, len(volumeMounts), 0, "no volume mounts present")

		//ensure that /licenses mount does not have any subPathExpr set
		for _, volumeMount := range volumeMounts {
			if volumeMount.MountPath == "/licenses" {
				assert.Equal(t, volumeMount.SubPathExpr, "", "subPathExpr is not empty")
				break
			}
		}
	}))
}

// TestNeo4jServicePortVariablesWithHTTPPortDisabled disables http port and check if the port is removed from loadbalancer service or not
func TestNeo4jServicePortVariablesWithHTTPPortDisabled(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.Services.Neo4j.Ports.HTTP.Enabled = false

		helmValues.Services.Neo4j.Ports.HTTPS.Enabled = true
		helmValues.Services.Neo4j.Ports.HTTPS.Port = 7473
		helmValues.Services.Neo4j.Ports.HTTPS.TargetPort = 8001
		helmValues.Services.Neo4j.Ports.HTTPS.Name = "https-port"

		helmValues.Services.Neo4j.Enabled = true

		helmValues.DisableLookups = true
		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, "no error should be seen while checking for neo4j lb service ports configuration with http port disabled")
		lbServiceName := fmt.Sprintf("%s-neo4j", model.DefaultHelmTemplateReleaseName)
		lbService := manifests.OfTypeWithName(&v1.Service{}, lbServiceName)
		assert.NotNil(t, lbService, fmt.Sprintf("no loadbalancer service found with name %s", lbServiceName))

		ports := lbService.(*v1.Service).Spec.Ports
		assert.Len(t, ports, 2, "there should be only 3 ports in the lbservice")
		for _, port := range ports {
			assert.NotEqual(t, port.Name, "http-port", "http-port is not expected")
			assert.NotEqual(t, port.Port, int32(7474), "7474 is not expected")
			assert.NotEqual(t, port.TargetPort, "7474", "7474 target port is not expected")
			switch port.Name {
			case "https-port":
				assert.Equal(t, int(port.Port), 7473, "https port should be 7473")
				assert.Equal(t, int(port.TargetPort.IntVal), 8001, "https target port should be 8001")
				continue
			}
		}
	}))
}

// TestNeo4jStatefulSetPortWithHTTPDisabled disables http port and check if the port is removed from statefulset service or not
func TestNeo4jStatefulSetPortWithHTTPDisabled(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		config := make(map[string]string, 1)
		config["dbms.connector.http.enabled"] = "false"
		helmValues.Config = config

		helmValues.DisableLookups = true
		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, "no error should be seen while checking for statefulset ports  with http port disabled")
		statefulset := manifests.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
		assert.NotNil(t, statefulset, "no statefulset found")
		ports := statefulset.(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0].Ports
		if edition == "community" {
			assert.Len(t, ports, 1, "only two statefulset ports should be present")
		} else {
			//enterprise has backup port enabled
			assert.Len(t, ports, 2, "only three statefulset ports should be present")
		}

		for _, port := range ports {
			assert.NotEqual(t, port.Name, "http", "http is not expected")
			assert.NotEqual(t, port.ContainerPort, int32(7474), "7474 is not expected")
		}

	}))
}

// TestNeo4jServicePortVariablesWithHTTPPortDisabledViaConfig disables http port via config and check if the port is removed from k8s service or not
func TestNeo4jServicePortVariablesWithHTTPPortDisabledViaConfig(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.Services.Neo4j.Ports.HTTP.Enabled = false
		helmValues.Services.Neo4j.Ports.HTTPS.Enabled = false
		helmValues.Services.Neo4j.Enabled = true
		helmValues.DisableLookups = true
		config := make(map[string]string, 1)
		config["dbms.connector.http.enabled"] = "false"
		helmValues.Config = config
		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, "no error should be seen while checking for neo4j lb service ports configuration with http port disabled")

		services := manifests.OfType(&v1.Service{})
		assert.NotNil(t, services, "no services found with name")

		for _, service := range services {
			ports := service.(*v1.Service).Spec.Ports
			for _, port := range ports {
				assert.NotEqual(t, port.Name, "http-port", "http-port is not expected")
				assert.NotEqual(t, port.Port, int32(7474), "7474 is not expected")
				assert.NotEqual(t, port.TargetPort, "7474", "7474 target port is not expected")
			}
		}

	}))
}

// TestNeo4jServicePortVariablesWithHTTPSPortDisabledViaConfig disables https port via config and check if the port is removed from k8s service or not
func TestNeo4jServicePortVariablesWithHTTPSPortDisabledViaConfig(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.Services.Neo4j.Enabled = true
		helmValues.DisableLookups = true
		config := make(map[string]string, 1)
		config["dbms.connector.https.enabled"] = "false"
		helmValues.Config = config
		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, "no error should be seen while checking for neo4j lb service ports configuration with http port disabled")

		services := manifests.OfType(&v1.Service{})
		assert.NotNil(t, services, "no services found with name")

		for _, service := range services {
			ports := service.(*v1.Service).Spec.Ports
			for _, port := range ports {
				assert.NotEqual(t, port.Name, "https-port", "https-port is not expected")
				assert.NotEqual(t, port.Port, int32(7473), "7473 is not expected")
				assert.NotEqual(t, port.TargetPort, "7473", "7473 target port is not expected")

			}
		}

	}))
}

// TestPodDisruptionBudgetLabelsAndValues checks if podDisruptionBudget has the provided labels and minAvai and maxUnavai values set or not
func TestPodDisruptionBudgetLabelsAndValues(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.PodDisruptionBudget.Enabled = true
		helmValues.PodDisruptionBudget.Labels = map[string]string{
			"demo": "neo4j",
		}
		helmValues.PodDisruptionBudget.MaxUnavailable = "5"
		helmValues.PodDisruptionBudget.MinAvailable = "3"
		helmValues.PodDisruptionBudget.MatchLabels = map[string]string{
			"name": "pdb",
		}
		matchExpressions := []model.MatchExpressions{
			{
				Key:      "demo",
				Operator: "Equals",
				Values:   []string{"neo4j"},
			},
		}
		helmValues.PodDisruptionBudget.MatchExpressions = matchExpressions

		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, fmt.Sprintf("error seen while testing pod distruption budget labels"))
		podDisruptionBudgets := manifests.OfType(&v12.PodDisruptionBudget{})
		assert.Len(t, podDisruptionBudgets, 1, "more than 1 podDisruptionBudget found")
		podDisruptionBudget := podDisruptionBudgets[0]
		assert.Contains(t, podDisruptionBudget.(*v12.PodDisruptionBudget).Labels, "demo", "missing label 'demo'")
		assert.Equal(t, podDisruptionBudget.(*v12.PodDisruptionBudget).Spec.MinAvailable.String(), helmValues.PodDisruptionBudget.MinAvailable)
		assert.Equal(t, podDisruptionBudget.(*v12.PodDisruptionBudget).Spec.MaxUnavailable.String(), helmValues.PodDisruptionBudget.MaxUnavailable)
		assert.Contains(t, podDisruptionBudget.(*v12.PodDisruptionBudget).Spec.Selector.MatchLabels, "name", "missing label 'name'")
		assert.Len(t, podDisruptionBudget.(*v12.PodDisruptionBudget).Spec.Selector.MatchExpressions, 1, "more than 1 matchExpressions found")
		for _, matchExpr := range podDisruptionBudget.(*v12.PodDisruptionBudget).Spec.Selector.MatchExpressions {
			assert.Equal(t, string(matchExpr.Operator), matchExpressions[0].Operator)
			assert.Equal(t, matchExpr.Key, matchExpressions[0].Key)
			assert.Equal(t, matchExpr.Values, matchExpressions[0].Values)
		}
	}))
}

// TestPodDisruptionBudgetLabelsAndValues checks if podDisruptionBudget has the provided labels and minAvai and maxUnavai values set or not
func TestEmptyPodDisruptionBudgetWhenDisabled(t *testing.T) {
	t.Parallel()
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.PodDisruptionBudget.Enabled = false
		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, fmt.Sprintf("error seen while testing pod distruption budget labels"))
		podDisruptionBudgets := manifests.OfType(&v12.PodDisruptionBudget{})
		assert.Nil(t, podDisruptionBudgets, "podDisruptionBudget found")
	}))
}

// TestServiceMonitorWhenEnabled checks for service monitor spec when enabled
func TestServiceMonitorWhenEnabled(t *testing.T) {
	t.Parallel()
	//assert.NoError(t, installPrometheusCRDS(), "error seen while installing prometheus crds")
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.ServiceMonitor.Enabled = true
		helmValues.ServiceMonitor.Port = "2204"
		helmValues.ServiceMonitor.Labels = map[string]string{
			"name": "service-monitor",
		}
		helmValues.ServiceMonitor.Interval = "30s"
		helmValues.ServiceMonitor.JobLabel = "name"
		helmValues.ServiceMonitor.Path = "/metrics"
		helmValues.ServiceMonitor.NamespaceSelector = model.NamespaceSelector{
			Any:        false,
			MatchNames: []string{"demo", "default"},
		}
		helmValues.ServiceMonitor.TargetLabels = []string{"name"}

		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, fmt.Sprintf("error seen while testing service monitor when enabled"))
		serviceMonitors := manifests.OfType(&pv1.ServiceMonitor{})
		assert.Len(t, serviceMonitors, 1, "serviceMonitor not found")
		serviceMonitor := serviceMonitors[0].(*pv1.ServiceMonitor)
		assert.Len(t, serviceMonitor.Spec.Endpoints, 1, "more than one endpoint found in service monitor")
		endPoint := serviceMonitor.Spec.Endpoints[0]
		assert.Equal(t, endPoint.Port, helmValues.ServiceMonitor.Port)
		assert.Equal(t, endPoint.Path, helmValues.ServiceMonitor.Path)
		assert.Equal(t, string(endPoint.Interval), helmValues.ServiceMonitor.Interval)
		assert.Equal(t, serviceMonitor.Labels, helmValues.ServiceMonitor.Labels)
		assert.Equal(t, serviceMonitor.Spec.TargetLabels, helmValues.ServiceMonitor.TargetLabels)
		assert.Equal(t, serviceMonitor.Spec.NamespaceSelector.MatchNames, helmValues.ServiceMonitor.NamespaceSelector.MatchNames)
	}))
}

// TestServiceMonitorWhenDisabled checks for service monitor spec when disabled
func TestServiceMonitorWhenDisabled(t *testing.T) {
	t.Parallel()
	//assert.NoError(t, installPrometheusCRDS(), "error seen while installing prometheus crds")
	var helmValues model.HelmValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues = model.DefaultCommunityValues
		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.ServiceMonitor.Enabled = false

		manifests, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		assert.NoError(t, err, fmt.Sprintf("error seen while testing service monitor when disabled"))
		serviceMonitors := manifests.OfType(&pv1.ServiceMonitor{})
		assert.Nil(t, serviceMonitors)

	}))
}

// TestNeo4jAuthPathVolumeMountWithPasswordFromSecret ensure that the neo4j-auth volume mount via passswordFromSecret is present
func TestNeo4jAuthPathVolumeMountWithPasswordFromSecret(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.Neo4J.PasswordFromSecret = "demo-secret"
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		statefulSet := manifest.OfType(&appsv1.StatefulSet{})[0]
		assert.NotNil(t, statefulSet, "statefulset missing")

		volumes := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Volumes
		assert.NotEqual(t, len(volumes), 0, "no volumes found")
		var names []string
		for _, volume := range volumes {
			names = append(names, volume.Name)
		}
		assert.Contains(t, names, "neo4j-auth", "missing neo4j-auth volume")

		volumeMounts := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0].VolumeMounts
		assert.NotEqual(t, len(volumeMounts), 0, "no volume mounts found")
		names = []string{}
		var mountPath []string
		for _, volumeMount := range volumeMounts {
			names = append(names, volumeMount.Name)
			mountPath = append(mountPath, volumeMount.MountPath)
		}
		assert.Contains(t, names, "neo4j-auth", "missing neo4j-auth volume mount")
		assert.Contains(t, mountPath, "/config/neo4j-auth", "missing neo4j-auth volume mount path")
	}))
}

// TestNeo4jAuthPathVolumeMountWithoutPasswordFromSecret ensure that the neo4j-auth volume mount without passwordFromSecret is present
func TestNeo4jAuthPathVolumeMountWithoutPasswordFromSecret(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		statefulSet := manifest.OfType(&appsv1.StatefulSet{})[0]
		assert.NotNil(t, statefulSet, "statefulset missing")

		volumes := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Volumes
		assert.NotEqual(t, len(volumes), 0, "no volumes found")
		var names []string
		for _, volume := range volumes {
			names = append(names, volume.Name)
		}
		assert.Contains(t, names, "neo4j-auth", "missing neo4j-auth volume")

		volumeMounts := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0].VolumeMounts
		assert.NotEqual(t, len(volumeMounts), 0, "no volume mounts found")
		names = []string{}
		var mountPath []string
		for _, volumeMount := range volumeMounts {
			names = append(names, volumeMount.Name)
			mountPath = append(mountPath, volumeMount.MountPath)
		}
		assert.Contains(t, names, "neo4j-auth", "missing neo4j-auth volume mount")
		assert.Contains(t, mountPath, "/config/neo4j-auth", "missing neo4j-auth volume mount path")
	}))
}

// TestNeo4jPodSpecDNSPolicy ensure that the provided dns policy is being set
func TestNeo4jPodSpecDNSPolicy(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.PodSpec.DNSPolicy = "demo"
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		statefulSet := manifest.OfType(&appsv1.StatefulSet{})[0]
		assert.NotNil(t, statefulSet, "statefulset missing")

		dnsPolicy := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.DNSPolicy
		assert.Equal(t, string(dnsPolicy), "demo", "dns policy not matching")
	}))
}

// TestNeo4jPodSpecTopologySpreadConstraints ensure that the provided topologySpreadConstraints are being set
func TestNeo4jPodSpecTopologySpreadConstraints(t *testing.T) {

	t.Parallel()
	tsc := model.TopologySpreadConstraint{
		MaxSkew:           1,
		TopologyKey:       "demo",
		WhenUnsatisfiable: "demo",
	}

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.PodSpec.TopologySpreadConstraints = []model.TopologySpreadConstraint{tsc}
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		statefulSet := manifest.OfType(&appsv1.StatefulSet{})[0]
		assert.NotNil(t, statefulSet, "statefulset missing")

		tscs := statefulSet.(*appsv1.StatefulSet).Spec.Template.Spec.TopologySpreadConstraints
		assert.Equal(t, len(tscs), 1, "more than 1 or zero topology spread constraints found")
		assert.Equal(t, tscs[0].MaxSkew, int32(1), "maxSkew not matching")
		assert.Equal(t, tscs[0].TopologyKey, "demo", "topologyKey not matching")
	}))
}

// checkMemoryResources runs helm template on all charts of all editions with invalid memory values
func checkMemoryResources(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string, memorySlice []string, containsErrMsg string) {

	var args []string
	setEdition := []string{"--set", fmt.Sprintf("neo4j.edition=%s", edition)}
	args = append(args, setEdition...)
	args = append(args, useDataModeAndAcceptLicense...)
	for _, memory := range memorySlice {
		setMemoryResource := []string{"--set", fmt.Sprintf("neo4j.resources.memory=%s", memory)}
		_, err := model.HelmTemplate(t, chart, args, setMemoryResource...)
		if !assert.Error(t, err) {
			return
		}
		if !assert.Contains(t, err.Error(), containsErrMsg) {
			return
		}
	}
}

// checkCPUResources runs helm template on all charts of all editions with invalid cpu values
func checkCPUResources(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string, cpuSlice []string, containsErrMsg string) {

	var args []string
	setEdition := []string{"--set", fmt.Sprintf("neo4j.edition=%s", edition)}
	args = append(args, setEdition...)
	args = append(args, useDataModeAndAcceptLicense...)
	for _, cpu := range cpuSlice {
		setCPUResource := []string{"--set", fmt.Sprintf("neo4j.resources.cpu=%s", cpu)}
		_, err := model.HelmTemplate(t, chart, args, setCPUResource...)
		if !assert.Error(t, err) {
			return
		}
		if !assert.Contains(t, err.Error(), containsErrMsg) {
			return
		}
	}
}

// checkCPUResources runs helm template on all charts of all editions with invalid cpu values
func checkResourcesAndLimits(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string, testCase Neo4jResourceTestCase) {

	var args []string
	setEdition := []string{"--set", fmt.Sprintf("neo4j.edition=%s", edition)}
	args = append(args, setEdition...)
	args = append(args, useDataModeAndAcceptLicense...)

	manifest, err := model.HelmTemplate(t, chart, args, testCase.arguments...)
	if !assert.NoError(t, err) {
		return
	}

	statefulSets := manifest.OfType(&appsv1.StatefulSet{})

	// should contain exactly one StatefulSet
	assert.Len(t, statefulSets, 1)

	statefulSet := statefulSets[0].(*appsv1.StatefulSet)
	// should contain exactly one Container
	assert.Len(t, statefulSet.Spec.Template.Spec.Containers, 1)
	container := statefulSet.Spec.Template.Spec.Containers[0]

	assert.Equal(t, container.Resources.Requests.Memory().String(), testCase.memory)
	assert.Equal(t, container.Resources.Limits.Memory().String(), testCase.memory)

	var cpuValue float64
	if strings.Contains(testCase.cpu, "m") {
		cpu := strings.Replace(testCase.cpu, "m", "", 1)
		cpuValue, err = strconv.ParseFloat(cpu, 64)
		if !assert.NoError(t, err) {
			return
		}
		cpuValue = cpuValue / 1000
	} else {
		cpuValue, err = strconv.ParseFloat(testCase.cpu, 64)
		if !assert.NoError(t, err) {
			return
		}
	}
	//if the test case cpu is a decimal value use AsDec()
	if strings.Contains(testCase.cpu, ".") {
		assert.Equal(t, container.Resources.Requests.Cpu().AsDec().String(), fmt.Sprintf("%g", cpuValue))
		assert.Equal(t, container.Resources.Limits.Cpu().AsDec().String(), fmt.Sprintf("%g", cpuValue))
	} else {
		t.Logf("checking %s == %s", container.Resources.Requests.Cpu().String(), fmt.Sprintf("%g", cpuValue))
		assert.Equal(t, container.Resources.Requests.Cpu().String(), testCase.cpu)
		assert.Equal(t, container.Resources.Limits.Cpu().String(), testCase.cpu)
	}

}

func checkNeo4jManifest(t *testing.T, manifest *model.K8sResources) {
	// should contain exactly one StatefulSet
	assert.Len(t, manifest.OfType(&appsv1.StatefulSet{}), 1)

	assertOnlyNeo4jImagesUsed(t, manifest)

	assertThreeServices(t, manifest)

	assertFourConfigMaps(t, manifest)
}

func assertFourConfigMaps(t *testing.T, manifest *model.K8sResources) {
	services := manifest.OfType(&v1.ConfigMap{})
	assert.Len(t, services, 4)
}

func assertThreeServices(t *testing.T, manifest *model.K8sResources) {
	services := manifest.OfType(&v1.Service{})
	assert.Len(t, services, 3)
}

func assertOnlyNeo4jImagesUsed(t *testing.T, manifest *model.K8sResources) {
	for _, neo4jStatefulSet := range manifest.OfType(&appsv1.StatefulSet{}) {
		assertOnlyNeo4jImagesUsedInStatefulSet(t, neo4jStatefulSet.(*appsv1.StatefulSet))
	}
	//TODO: add checks on Pods, Jobs, CronJobs, ReplicaSets, Deployments and anything else that can contain an image
}

func assertOnlyNeo4jImagesUsedInStatefulSet(t *testing.T, neo4jStatefulSet *appsv1.StatefulSet) {
	for _, container := range neo4jStatefulSet.Spec.Template.Spec.Containers {
		assert.Contains(t, container.Image, "neo4j:")
	}

	for _, container := range neo4jStatefulSet.Spec.Template.Spec.InitContainers {
		assert.Contains(t, container.Image, "neo4j:")
	}
}

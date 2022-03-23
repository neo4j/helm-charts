package unit_tests

import (
	"errors"
	"fmt"

	"github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
func forEachPrimaryChart(t *testing.T, test func(*testing.T, model.Neo4jHelmChart)) {
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart) {
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
func forEachSupportedEdition(t *testing.T, chart model.Neo4jHelmChart, test func(*testing.T, model.Neo4jHelmChart, string)) {
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

func andEachSupportedEdition(test func(*testing.T, model.Neo4jHelmChart, string)) func(t *testing.T, chart model.Neo4jHelmChart) {
	return func(t *testing.T, chart model.Neo4jHelmChart) {
		forEachSupportedEdition(t, chart, func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
			test(t, chart, edition)
		})
	}
}

func TestErrorThrownIfNoDataVolumeModeChosen(t *testing.T) {
	t.Parallel()
	forEachPrimaryChart(t, andEachSupportedEdition(
		func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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
		func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, testCase []string) {
		t.Parallel()
		_, err := model.HelmTemplate(t, chart, testCase)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "to use Neo4j Enterprise Edition you must have a Neo4j license agreement")
		assert.Contains(t, err.Error(), "Set neo4j.acceptLicenseAgreement: \"yes\" to confirm that you have a Neo4j license agreement.")
	}

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChart) {
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

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, testCase []string) {
		t.Parallel()
		manifest, err := model.HelmTemplate(t, chart, requiredDataMode, testCase...)
		if !assert.NoError(t, err) {
			return
		}

		checkNeo4jManifest(t, manifest)
	}

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChart) {
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

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, testCase []string) {
		t.Parallel()
		manifest, err := model.HelmTemplate(t, chart, requiredDataMode, testCase...)
		if !assert.NoError(t, err) {
			return
		}

		checkNeo4jManifest(t, manifest)
	}

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChart) {
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

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChart) {
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

	forEachPrimaryChart(t, func(t *testing.T, chart model.Neo4jHelmChart) {
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

func TestAdditionalEnvVars(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(
		func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, s string) {
		_, err := model.HelmTemplateFromYamlFile(t, chart, resources.BoolsInConfig, acceptLicenseAgreement...)
		assert.Error(t, err, "Helm chart should fail if config contains boolean values")
		assert.Contains(t, err.Error(), "config values must be strings.")
		assert.Contains(t, err.Error(), "metrics.enabled")
		assert.Contains(t, err.Error(), "type: bool")
	}))
}

func TestIntsInConfig(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, s string) {
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

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, s string) {
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

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, s string) {
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

// Tests the "base" helm command used for Integration Tests
func TestBaseHelmTemplate(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		diskName := model.DefaultHelmTemplateReleaseName.DiskName()
		_, err := model.RunHelmCommand(t, model.BaseHelmCommand("template", &model.DefaultHelmTemplateReleaseName, chart, edition, &diskName))
		if !assert.NoError(t, err) {
			return
		}
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
		{&neo4jDotName, false, &model.DefaultPassword, authSecretExpectation{helmFailsWithError: errors.New("unsupported State: Cannot set neo4j.password when Neo4j authis disabled (dbms.security.auth_enabled=false). Either remove neo4j.password setting or enable Neo4j auth")}},
		{nil, false, &model.DefaultPassword, authSecretExpectation{helmFailsWithError: errors.New("unsupported State: Cannot set neo4j.password when Neo4j authis disabled (dbms.security.auth_enabled=false). Either remove neo4j.password setting or enable Neo4j auth")}},
	}

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, edition string, testCase authSecretTest) {
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

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		for i, testCase := range testCases {
			t.Run(fmt.Sprintf("%d %s", i, testCase), func(t *testing.T) {
				doTestCase(t, chart, edition, testCase)
			})
		}
	}))
}

func TestDefaultLabels(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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

func TestExtraLabels(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		_, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.EmptyImageCredentials.HelmArgs()...)
		if !assert.Error(t, err) {
			return
		}
		if !assert.Contains(t, err.Error(), "Username field cannot be empty") {
			return
		}
		if !assert.Contains(t, err.Error(), "Password field cannot be empty") {
			return
		}
		if !assert.Contains(t, err.Error(), "Email field cannot be empty") {
			return
		}
		if !assert.Contains(t, err.Error(), "name field cannot be empty") {
			return
		}
	}))
}

func TestDuplicateImageCredentials(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		_, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.DuplicateImageCredentials.HelmArgs()...)
		if !assert.Error(t, err) {
			return
		}
		if !assert.Contains(t, err.Error(), "Duplicate \"names\" found in imageCredentials list. Please remove duplicates") {
			return
		}
	}))
}

func TestMissingImageCredentials(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		_, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense, resources.MissingImageCredentials.HelmArgs()...)
		if !assert.Error(t, err) {
			return
		}
		if !assert.Contains(t, err.Error(), "Missing imageCredential entry") {
			return
		}
	}))
}

//TestEmptyImagePullSecrets ensures empty imagePullSecret names or names with just spaces are not included in the cluster formation
func TestEmptyImagePullSecrets(t *testing.T) {
	t.Parallel()

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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

func TestErrorIsThrownForInvalidMemoryResources(t *testing.T) {

	t.Parallel()
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		invalidMemory := []string{
			"",
		}
		checkMemoryResources(t, chart, edition, invalidMemory, "neo4j.resources.memory cannot be set to \"\"")
	}
	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestErrorIsThrownForEmptyCPUResources(t *testing.T) {

	t.Parallel()
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		invalidCPU := []string{
			"",
		}
		checkCPUResources(t, chart, edition, invalidCPU, "neo4j.resources.cpu cannot be set to \"\"")
	}
	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestErrorIsThrownForInvalidCPUResourcesRegex(t *testing.T) {

	t.Parallel()
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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
	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
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

	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		for i, testCase := range testCases {
			t.Run(fmt.Sprintf("%d %s", i, testCase), func(t *testing.T) {
				checkResourcesAndLimits(t, chart, edition, testCase)
			})
		}
	}))
}

//checkMemoryResources runs helm template on all charts of all editions with invalid memory values
func checkMemoryResources(t *testing.T, chart model.Neo4jHelmChart, edition string, memorySlice []string, containsErrMsg string) {

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

//checkCPUResources runs helm template on all charts of all editions with invalid cpu values
func checkCPUResources(t *testing.T, chart model.Neo4jHelmChart, edition string, cpuSlice []string, containsErrMsg string) {

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

//checkCPUResources runs helm template on all charts of all editions with invalid cpu values
func checkResourcesAndLimits(t *testing.T, chart model.Neo4jHelmChart, edition string, testCase Neo4jResourceTestCase) {

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

package unit_tests

import (
	"bufio"
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"os"
	"strings"
	"testing"
)

func containsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Very quick test to check that no errors are thrown and a couple of values from the default neo4j conf show up
func TestPopulateFromFile(t *testing.T) {
	testCases := []string{
		"enterprise",
		"community",
	}

	edition, found := os.LookupEnv("NEO4J_EDITION")
	if found && !containsString(testCases, edition) {
		testCases = append(testCases, edition)
	}

	doTestCase := func(t *testing.T, edition string) {
		t.Parallel()
		conf, err := (&model.Neo4jConfiguration{}).PopulateFromFile(fmt.Sprintf("neo4j/neo4j-%s.conf", edition))
		if !assert.NoError(t, err) {
			return
		}

		value, found := conf.Conf()["server.windows_service_name"]
		assert.True(t, found)
		assert.Equal(t, "neo4j", value)

		_, jvmKeyFound := conf.Conf()["server.jvm.additional"]
		assert.False(t, jvmKeyFound)

		//TODO: This is to be enabled in 5.0
		//value, found = conf.Conf()["dbms.logs.default_format"]
		//assert.True(t, found)
		//assert.Equal(t, "JSON", value)

		assert.Contains(t, conf.JvmArgs(), "-XX:+UnlockDiagnosticVMOptions")
		assert.Contains(t, conf.JvmArgs(), "-XX:+DebugNonSafepoints")
		assert.Greater(t, len(conf.JvmArgs()), 1)
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			doTestCase(t, testCase)
		})
	}
}

func TestJvmAdditionalConfig(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		aditionalJvmArgs := []string{
			"-XX:+HeapDumpOnOutOfMemoryError",
			"-XX:HeapDumpPath=./java_pid<pid>.hprof",
			"-XX:+UseGCOverheadLimit",
			"-XX:MaxMetaspaceSize=180m",
			"-XX:ReservedCodeCacheSize=40m",
		}
		helmValues := model.DefaultEnterpriseValues
		helmValues.Jvm.UseNeo4JDefaultJvmArguments = true
		helmValues.Jvm.AdditionalJvmArguments = aditionalJvmArgs
		manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)
		if !assert.NoError(t, err) {
			return
		}

		userConfigMap := manifest.OfTypeWithName(&v1.ConfigMap{}, model.DefaultHelmTemplateReleaseName.UserConfigMapName()).(*v1.ConfigMap)
		assert.Contains(t, userConfigMap.Data["server.jvm.additional"], "-XX:+HeapDumpOnOutOfMemoryError")
		assert.Contains(t, userConfigMap.Data["server.jvm.additional"], "-XX:HeapDumpPath=./java_pid<pid>.hprof")
		assert.Contains(t, userConfigMap.Data["server.jvm.additional"], "-XX:+UseGCOverheadLimit")
		assert.Contains(t, userConfigMap.Data["server.jvm.additional"], "-XX:MaxMetaspaceSize=180m")
		assert.Contains(t, userConfigMap.Data["server.jvm.additional"], "-XX:ReservedCodeCacheSize=40m")

		err = checkConfigMapContainsJvmAdditionalFromDefaultConf(t, edition, userConfigMap)
		if !assert.NoError(t, err) {
			return
		}

		checkNeo4jManifest(t, manifest, 3)
	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestMetaspaceConfigs(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		metaSpaceConfig := make(map[string]string)
		metaSpaceConfig["server.memory.pagecache.size"] = "74m"
		metaSpaceConfig["server.memory.heap.initial_size"] = "317m"
		metaSpaceConfig["server.memory.heap.max_size"] = "317m"
		helmValues := model.DefaultEnterpriseValues
		helmValues.Config = metaSpaceConfig
		manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)

		if !assert.NoError(t, err) {
			return
		}

		userConfigMap := manifest.OfTypeWithName(&v1.ConfigMap{}, model.DefaultHelmTemplateReleaseName.UserConfigMapName()).(*v1.ConfigMap)
		assert.Equal(t, userConfigMap.Data["server.memory.heap.initial_size"], "317m")
		assert.Equal(t, userConfigMap.Data["server.memory.heap.max_size"], "317m")
		assert.Equal(t, userConfigMap.Data["server.memory.pagecache.size"], "74m")

	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestFullnameOverrideStatefulSet(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues := model.DefaultEnterpriseValues
		fullNameOverride := "use-this-name-instead"
		helmValues.FullnameOverride = fullNameOverride
		helmValues.Neo4J.Edition = edition
		manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)

		if !assert.NoError(t, err) {
			return
		}
		neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		assert.Equal(t, neo4jStatefulSet.GetName(), fullNameOverride)
		assert.Equal(t, neo4jStatefulSet.Labels["helm.neo4j.com/instance"], fullNameOverride)
		assert.Equal(t, neo4jStatefulSet.Spec.ServiceName, fullNameOverride)
		assert.Equal(t, neo4jStatefulSet.Spec.Selector.MatchLabels["helm.neo4j.com/instance"], fullNameOverride)
		assert.Equal(t, neo4jStatefulSet.Spec.Template.ObjectMeta.Labels["helm.neo4j.com/instance"], fullNameOverride)
		assert.Contains(t, neo4jStatefulSet.Spec.Template.ObjectMeta.Annotations, fmt.Sprintf("checksum/%s-config", fullNameOverride))
		neo4jContainer := neo4jStatefulSet.Spec.Template.Spec.Containers[0]
		assert.Contains(t, neo4jContainer.EnvFrom, v1.EnvFromSource{
			ConfigMapRef: &v1.ConfigMapEnvSource{
				LocalObjectReference: v1.LocalObjectReference{Name: fmt.Sprintf("%s-env", fullNameOverride)},
			},
		})
		assert.Contains(t, neo4jContainer.Env, v1.EnvVar{
			Name:  "SERVICE_NEO4J_ADMIN",
			Value: fmt.Sprintf("%s-admin.neo4j-my-release.svc.cluster.local", fullNameOverride),
		})
		assert.Contains(t, neo4jContainer.Env, v1.EnvVar{
			Name:  "SERVICE_NEO4J_INTERNALS",
			Value: fmt.Sprintf("%s-internals.neo4j-my-release.svc.cluster.local", fullNameOverride),
		})
		assert.Contains(t, neo4jContainer.Env, v1.EnvVar{
			Name:  "SERVICE_NEO4J",
			Value: fmt.Sprintf("%s.neo4j-my-release.svc.cluster.local", fullNameOverride),
		})
	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestNameOverrideStatefulSet(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		helmValues := model.DefaultEnterpriseValues
		helmValues.NameOverride = "use-this-name-instead"
		nameOverride := "my-release-use-this-name-instead"
		helmValues.Neo4J.Edition = edition
		manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)

		if !assert.NoError(t, err) {
			return
		}
		neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
		assert.Equal(t, neo4jStatefulSet.GetName(), nameOverride)
		assert.Equal(t, neo4jStatefulSet.Labels["helm.neo4j.com/instance"], nameOverride)
		assert.Equal(t, neo4jStatefulSet.Spec.ServiceName, nameOverride)
		assert.Equal(t, neo4jStatefulSet.Spec.Selector.MatchLabels["helm.neo4j.com/instance"], nameOverride)
		assert.Equal(t, neo4jStatefulSet.Spec.Template.ObjectMeta.Labels["helm.neo4j.com/instance"], nameOverride)
		assert.Contains(t, neo4jStatefulSet.Spec.Template.ObjectMeta.Annotations, fmt.Sprintf("checksum/%s-config", nameOverride))
		neo4jContainer := neo4jStatefulSet.Spec.Template.Spec.Containers[0]
		assert.Contains(t, neo4jContainer.EnvFrom, v1.EnvFromSource{
			ConfigMapRef: &v1.ConfigMapEnvSource{
				LocalObjectReference: v1.LocalObjectReference{Name: fmt.Sprintf("%s-env", nameOverride)},
			},
		})
		assert.Contains(t, neo4jContainer.Env, v1.EnvVar{
			Name:  "SERVICE_NEO4J_ADMIN",
			Value: fmt.Sprintf("%s-admin.neo4j-my-release.svc.cluster.local", nameOverride),
		})
		assert.Contains(t, neo4jContainer.Env, v1.EnvVar{
			Name:  "SERVICE_NEO4J_INTERNALS",
			Value: fmt.Sprintf("%s-internals.neo4j-my-release.svc.cluster.local", nameOverride),
		})
		assert.Contains(t, neo4jContainer.Env, v1.EnvVar{
			Name:  "SERVICE_NEO4J",
			Value: fmt.Sprintf("%s.neo4j-my-release.svc.cluster.local", nameOverride),
		})
	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func checkConfigMapContainsJvmAdditionalFromDefaultConf(t *testing.T, edition string, userConfigMap *v1.ConfigMap) error {
	// check that we picked up jvm additional from the conf file
	file, err := os.Open(fmt.Sprintf("neo4j/neo4j-%s.conf", edition))
	defer file.Close()
	if err != nil {
		return err
	}

	n := 0
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		var line = scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "server.jvm.additional") {
			line = strings.Replace(line, "server.jvm.additional=", "", 1)
			assert.Contains(t, userConfigMap.Data["server.jvm.additional"], line)
			n++
		}
		if err != nil {
			return err
		}

	}
	// The conf file should contain at least 4 (this just sanity checks that the scanner and string handling stuff above didn't screw up)
	assert.GreaterOrEqual(t, n, 4)
	return nil
}

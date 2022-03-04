package unit_tests

import (
	"bufio"
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
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
		conf, err := (&model.Neo4jConfiguration{}).PopulateFromFile(fmt.Sprintf("neo4j-standalone/neo4j-%s.conf", edition))
		if !assert.NoError(t, err) {
			return
		}

		value, found := conf.Conf()["dbms.windows_service_name"]
		assert.True(t, found)
		assert.Equal(t, "neo4j", value)

		_, jvmKeyFound := conf.Conf()["dbms.jvm.additional"]
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

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense,
			"-f", "internal/resources/jvmAdditionalSettings.yaml",
			"--set", "neo4j.edition="+edition,
		)
		if !assert.NoError(t, err) {
			return
		}

		userConfigMap := manifest.OfTypeWithName(&v1.ConfigMap{}, model.DefaultHelmTemplateReleaseName.UserConfigMapName()).(*v1.ConfigMap)
		assert.Contains(t, userConfigMap.Data["dbms.jvm.additional"], "-XX:+HeapDumpOnOutOfMemoryError")
		assert.Contains(t, userConfigMap.Data["dbms.jvm.additional"], "-XX:HeapDumpPath=./java_pid<pid>.hprof")
		assert.Contains(t, userConfigMap.Data["dbms.jvm.additional"], "-XX:+UseGCOverheadLimit")
		assert.Contains(t, userConfigMap.Data["dbms.jvm.additional"], "-XX:MaxMetaspaceSize=180m")
		assert.Contains(t, userConfigMap.Data["dbms.jvm.additional"], "-XX:ReservedCodeCacheSize=40m")

		err = checkConfigMapContainsJvmAdditionalFromDefaultConf(t, edition, userConfigMap)
		if !assert.NoError(t, err) {
			return
		}

		checkNeo4jManifest(t, manifest)
	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func TestMetaspaceConfigs(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChart, edition string) {
		manifest, err := model.HelmTemplate(t, chart, useDataModeAndAcceptLicense,
			"-f", "internal/resources/metaspaceconfigs.yaml",
			"--set", "neo4j.edition="+edition,
		)
		if !assert.NoError(t, err) {
			return
		}

		userConfigMap := manifest.OfTypeWithName(&v1.ConfigMap{}, model.DefaultHelmTemplateReleaseName.UserConfigMapName()).(*v1.ConfigMap)
		assert.Equal(t, userConfigMap.Data["dbms.memory.heap.initial_size"], "317m")
		assert.Equal(t, userConfigMap.Data["dbms.memory.heap.max_size"], "317m")
		assert.Equal(t, userConfigMap.Data["dbms.memory.pagecache.size"], "74m")
		assert.Equal(t, userConfigMap.Data["dbms.query_cache_size"], "10")

	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

func checkConfigMapContainsJvmAdditionalFromDefaultConf(t *testing.T, edition string, userConfigMap *v1.ConfigMap) error {
	// check that we picked up jvm additional from the conf file
	file, err := os.Open(fmt.Sprintf("neo4j-standalone/neo4j-%s.conf", edition))
	defer file.Close()
	if err != nil {
		return err
	}

	n := 0
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		var line = scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "dbms.jvm.additional") {
			line = strings.Replace(line, "dbms.jvm.additional=", "", 1)
			assert.Contains(t, userConfigMap.Data["dbms.jvm.additional"], line)
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

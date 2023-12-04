package unit_tests

import (
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	"strings"
	"testing"
)

// TestAnalyticsConfigForPrimaryType checks the required config when analytics flag is enabled and type is primary
func TestAnalyticsConfigForPrimaryType(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.Analytics.Enabled = true
		helmValues.Analytics.Type.Name = "primary"
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		configMaps := manifest.OfType(&v1.ConfigMap{})
		assert.NotNil(t, configMaps, "statefulset missing")
		assert.Greaterf(t, len(configMaps), 0, "no configmaps found")

		for _, configMap := range configMaps {
			cm := configMap.(*v1.ConfigMap)
			if strings.Contains(cm.Name, "default-config") {
				assert.Contains(t, cm.Data, "initial.dbms.default_primaries_count")
				assert.Equal(t, cm.Data["initial.dbms.default_primaries_count"], "1")
			}

		}
	}))
}

// TestAnalyticsConfigForSecondaryType checks the required config when analytics flag is enabled and type is secondary
func TestAnalyticsConfigForSecondaryType(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.Analytics.Enabled = true
		helmValues.Analytics.Type.Name = "secondary"
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		configMaps := manifest.OfType(&v1.ConfigMap{})
		assert.NotNil(t, configMaps, "statefulset missing")
		assert.Greaterf(t, len(configMaps), 0, "no configmaps found")

		for _, configMap := range configMaps {
			cm := configMap.(*v1.ConfigMap)
			if strings.Contains(cm.Name, "default-config") {
				assert.Contains(t, cm.Data, "server.cluster.system_database_mode")
				assert.Contains(t, cm.Data, "dbms.cluster.minimum_initial_system_primaries_count")
				assert.Contains(t, cm.Data, "internal.db.cluster.raft.minimum_voting_members")
				assert.Contains(t, cm.Data, "dbms.security.procedures.unrestricted")
				assert.Contains(t, cm.Data, "dbms.security.http_auth_allowlist")
				assert.Equal(t, cm.Data["server.cluster.system_database_mode"], "SECONDARY")
			}

		}
	}))
}

// TestServiceAccountCreationWhenAnalyticsEnabled checks the serviceaccount creation when analytics is enabled
func TestServiceAccountCreationWhenAnalyticsEnabled(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.Analytics.Enabled = true
		helmValues.Analytics.Type.Name = "secondary"
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		serviceAccounts := manifest.OfType(&v1.ServiceAccount{})
		roles := manifest.OfType(&v12.Role{})
		roleBindings := manifest.OfType(&v12.RoleBinding{})
		assert.NotNil(t, serviceAccounts, "serviceaccount missing")
		assert.NotNil(t, roles, "roles missing")
		assert.NotNil(t, roleBindings, "rolebindings missing")
		assert.Len(t, serviceAccounts, 1)
		assert.Len(t, roles, 1)
		assert.Len(t, roleBindings, 1)

		roleBinding := roleBindings[0].(*v12.RoleBinding)
		assert.Equal(t, roleBinding.Name, fmt.Sprintf("%s-service-binding", model.DefaultHelmTemplateReleaseName))
		assert.Len(t, roleBinding.Subjects, 1)
		assert.Equal(t, roleBinding.Subjects[0].Name, string(model.DefaultHelmTemplateReleaseName))
		assert.Equal(t, roleBinding.RoleRef.Name, fmt.Sprintf("%s-service-reader", model.DefaultHelmTemplateReleaseName))
	}))
}

// TestAnalyticsConfigWhenDisabled checks that required config is not present when analytics config disabled
func TestAnalyticsConfigWhenDisabled(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.Analytics.Enabled = false

		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		configMaps := manifest.OfType(&v1.ConfigMap{})
		assert.NotNil(t, configMaps, "statefulset missing")
		assert.Greaterf(t, len(configMaps), 0, "no configmaps found")

		for _, configMap := range configMaps {
			cm := configMap.(*v1.ConfigMap)
			if strings.Contains(cm.Name, "default-config") {
				assert.NotContains(t, cm.Data, "initial.dbms.default_primaries_count")
			}

		}
	}))
}

// TestServiceAccountCreationWhenAnalyticsEnabled checks the k8s service creation when analytics is enabled
func TestInternalServiceCreationWhenAnalyticsEnabled(t *testing.T) {

	t.Parallel()

	helmValues := model.DefaultCommunityValues
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {

		if edition == "enterprise" {
			helmValues = model.DefaultEnterpriseValues
		}
		helmValues.DisableLookups = true
		helmValues.Analytics.Enabled = true
		helmValues.Analytics.Type.Name = "primary"
		manifest, err := model.HelmTemplateFromStruct(t, chart, helmValues, "--dry-run")
		if !assert.NoError(t, err) {
			return
		}

		services := manifest.OfType(&v1.Service{})
		assert.NotNil(t, services, "no services found")
		for _, service := range services {
			svc := service.(*v1.Service)
			if strings.Contains(svc.Name, "internals") {
				ports := svc.Spec.Ports
				var targetPorts []string
				for _, port := range ports {
					targetPorts = append(targetPorts, port.TargetPort.StrVal)
				}
				assert.Contains(t, targetPorts, "7688", "missing 7688 port")
				assert.Contains(t, targetPorts, "5000", "missing 5000 port")
				assert.Contains(t, targetPorts, "6000", "missing 6000 port")
				assert.Contains(t, targetPorts, "7000", "missing 7000 port")
			}
		}

	}))
}

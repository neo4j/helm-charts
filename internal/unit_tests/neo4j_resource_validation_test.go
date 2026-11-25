package unit_tests

import (
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

func TestNeo4jResourceValidation(t *testing.T) {
	t.Parallel()
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		tests := []struct {
			name   string
			values []string
			err    string
		}{
			{"default CPU minimum", []string{"neo4j.resources.cpu=100m"}, "less than minimum"},
			{"default memory minimum", []string{"neo4j.resources.memory=512Mi"}, "less than minimum"},
			{"explicit CPU minimum", []string{"neo4j.resourcesValidation.enabled=true", "neo4j.resources.cpu=100m"}, "less than minimum"},
			{"explicit memory minimum", []string{"neo4j.resourcesValidation.enabled=true", "neo4j.resources.memory=512Mi"}, "less than minimum"},
			{"disabled invalid CPU", []string{"neo4j.resourcesValidation.enabled=false", "neo4j.resources.cpu=invalid"}, "Invalid cpu value"},
			{"disabled invalid memory", []string{"neo4j.resourcesValidation.enabled=false", "neo4j.resources.requests.memory=invalid"}, "Invalid memory value"},
			{"disabled empty CPU", []string{"neo4j.resourcesValidation.enabled=false", "neo4j.resources.requests.cpu="}, "cannot be set to"},
			{"disabled empty memory", []string{"neo4j.resourcesValidation.enabled=false", "neo4j.resources.memory="}, "cannot be set to"},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				args := []string{"--set", "neo4j.name=resource-validation", "--set", "volumes.data.mode=defaultStorageClass", "--set", "neo4j.acceptLicenseAgreement=yes", "--set", "neo4j.edition=" + edition}
				for _, value := range test.values {
					args = append(args, "--set", value)
				}
				_, err := model.HelmTemplate(t, chart, args)
				require.ErrorContains(t, err, test.err)
			})
		}
	}))
}

func TestNeo4jResourceValidationDisabled(t *testing.T) {
	t.Parallel()
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		tests := []struct {
			name        string
			values      []string
			cpuLimit    string
			memoryLimit string
		}{
			{"shorthand", []string{"neo4j.resources.cpu=100m", "neo4j.resources.memory=512Mi"}, "100m", "512Mi"},
			{"mixed", []string{"neo4j.resources.requests.cpu=100m", "neo4j.resources.memory=512Mi"}, "100m", "512Mi"},
			{"requests only", []string{"neo4j.resources.requests.cpu=100m", "neo4j.resources.requests.memory=512Mi"}, "0", "0"},
			{"independent limits", []string{"neo4j.resources.requests.cpu=100m", "neo4j.resources.requests.memory=512Mi", "neo4j.resources.limits.cpu=1", "neo4j.resources.limits.memory=2Gi"}, "1", "2Gi"},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				args := []string{"--set", "neo4j.name=resource-validation", "--set", "volumes.data.mode=defaultStorageClass", "--set", "neo4j.acceptLicenseAgreement=yes", "--set", "neo4j.edition=" + edition, "--set", "neo4j.resourcesValidation.enabled=false"}
				for _, value := range test.values {
					args = append(args, "--set", value)
				}
				manifest, err := model.HelmTemplate(t, chart, args)
				require.NoError(t, err)
				sets := manifest.OfType(&appsv1.StatefulSet{})
				require.Len(t, sets, 1)
				resources := sets[0].(*appsv1.StatefulSet).Spec.Template.Spec.Containers[0].Resources
				require.Equal(t, "100m", resources.Requests.Cpu().String())
				require.Equal(t, "512Mi", resources.Requests.Memory().String())
				require.Equal(t, test.cpuLimit, resources.Limits.Cpu().String())
				require.Equal(t, test.memoryLimit, resources.Limits.Memory().String())
			})
		}
	}))
}

func TestNeo4jResourceValidationDoesNotChangePodTemplate(t *testing.T) {
	t.Parallel()
	forEachPrimaryChart(t, andEachSupportedEdition(func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		args := []string{"--set", "neo4j.name=resource-validation", "--set", "volumes.data.mode=defaultStorageClass", "--set", "neo4j.acceptLicenseAgreement=yes", "--set", "neo4j.edition=" + edition}
		baseline, err := model.HelmTemplate(t, chart, args)
		require.NoError(t, err)
		for _, enabled := range []string{"true", "false"} {
			manifest, err := model.HelmTemplate(t, chart, args, "--set", "neo4j.resourcesValidation.enabled="+enabled)
			require.NoError(t, err)
			require.Equal(t, baseline.OfType(&appsv1.StatefulSet{}), manifest.OfType(&appsv1.StatefulSet{}))
		}
	}))
}

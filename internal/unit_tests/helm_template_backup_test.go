package unit_tests

import (
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	"testing"
)

// TestBackupInstallationWithNoValues checks backup helm chart installation with no values
func TestBackupInstallationWithNoValues(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.Contains(t, err.Error(), "Empty fields. Please set databaseAdminServiceName")
}

// TestBackupValues checks backup helm chart with sample values
func TestBackupValues(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while trying to install helm backup")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)
	assert.Equal(t, cronjob.Spec.Schedule, "* * * * *", fmt.Sprintf("cronjob schedule %s does not match with * * * * *", cronjob.Spec.Schedule))
	containers := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1, "there should be only one container present")
	container := containers[0]

	for _, envVar := range container.Env {
		switch envVar.Name {
		case "DATABASE_SERVICE_NAME":
			assert.Equal(t, envVar.Value, helmValues.Backup.DatabaseAdminServiceName, fmt.Sprintf("database address service name %s not matching with %s", helmValues.Backup.DatabaseAdminServiceName, envVar.Value))
		case "CLOUD_PROVIDER":
			assert.Equal(t, envVar.Value, helmValues.Backup.CloudProvider, fmt.Sprintf("cloud provider %s not matching with %s", helmValues.Backup.CloudProvider, envVar.Value))
		case "DATABASE":
			assert.Equal(t, envVar.Value, helmValues.Backup.Database, fmt.Sprintf("backup database value %s not matching with %s", helmValues.Backup.Database, envVar.Value))
		}
	}
	podSecurityContext := cronjob.Spec.JobTemplate.Spec.Template.Spec.SecurityContext
	assert.Equal(t, *podSecurityContext.RunAsNonRoot, true, fmt.Sprintf("security context runAsNonRoot %v should be true", podSecurityContext.RunAsNonRoot))
	assert.Equal(t, int(*podSecurityContext.RunAsUser), 7474, fmt.Sprintf("security context runAsNonRoot %v should be 7474", *podSecurityContext.RunAsUser))
	assert.Equal(t, int(*podSecurityContext.RunAsGroup), 7474, fmt.Sprintf("security context runAsGroup %v should be 7474", *podSecurityContext.RunAsGroup))
}

// TestBackupPodLabelsAndAnnotations checks backup helm chart for labels and annotations
func TestBackupPodLabelsAndAnnotations(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"
	helmValues.Neo4J.Labels = map[string]string{
		"demo1": "key1",
	}
	helmValues.Neo4J.PodLabels = map[string]string{
		"demo2": "key2",
	}
	helmValues.Neo4J.PodAnnotations = map[string]string{
		"demo3": "key3",
	}
	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while trying to install helm backup")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)
	assert.Contains(t, cronjob.Labels, "demo1", "missing labels demo1")
	podLabels := cronjob.Spec.JobTemplate.Spec.Template.ObjectMeta.Labels
	assert.Contains(t, podLabels, "demo2", "missing podLabel demo2")
	podAnnotations := cronjob.Spec.JobTemplate.Spec.Template.ObjectMeta.Annotations
	assert.Contains(t, podAnnotations, "demo3", "missing podAnnotation demo3")
}

// TestBackupNameOverride checks backup helm chart with nameOverride
func TestBackupNameOverride(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"
	helmValues.NameOverride = "testbackup"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while trying to install helm backup")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)
	assert.Contains(t, cronjob.ObjectMeta.Name, helmValues.NameOverride, "missing nameoverride")
}

// TestBackupNameFullOverride checks backup helm chart with fullNameOverride
func TestBackupNameFullOverride(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"
	helmValues.FullnameOverride = "testbackup"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while trying to install helm backup")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)
	assert.Equal(t, cronjob.ObjectMeta.Name, helmValues.FullnameOverride, "missing fullNameOverride")
}

// TestBackupEmptySecretKeyName checks backup helm chart with fullNameOverride
func TestBackupEmptySecretKeyName(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.Error(t, err, "error must be seen while trying to install helm backup")
	assert.Contains(t, err.Error(), "Empty secretKeyName")
}

// TestBackupInvalidSecretName checks backup helm chart installation with a secret that does not exists
func TestBackupInvalidSecretName(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "demo1"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"

	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	_, err := helmClient.Install(t, "demo", "demo-ns", helmValues)
	assert.Contains(t, err.Error(), fmt.Sprintf("Secret %s configured in 'backup.secretname' not found", helmValues.Backup.SecretName))
}

// TestBackupEmptyServiceNameAndIPFields checks backup helm chart installation with empty service name and ip fields
func TestBackupEmptyServiceNameAndIPFields(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = ""
	helmValues.Backup.DatabaseAdminServiceIP = ""
	helmValues.Backup.Database = "neo4j1"
	helmValues.FullnameOverride = "testbackup"

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.Contains(t, err.Error(), "Empty fields", "error message should contain empty fields")
}

// TestBackupNodeSelectorLabelsWithDisableLookups checks nodeSelector labels with disableLookups set to true
func TestBackupNodeSelectorLabelsWithDisableLookups(t *testing.T) {
	t.Parallel()

	nodeSelectorLabels := map[string]string{
		"label1": "value1",
		"label2": "value2",
	}
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.NodeSelector = nodeSelectorLabels
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.Database = "neo4j1"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing helm template on backup helm chart with disableLookups enabled and nodeselector labels ")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)
	assert.Equal(t, cronjob.Spec.JobTemplate.Spec.Template.Spec.NodeSelector, nodeSelectorLabels, "nodeSelector Labels not matching")
}

// TestNeo4jBackupPodTolerations checks for tolerations in the backup cronjob
func TestNeo4jBackupPodTolerations(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	dummyToleration := model.Toleration{
		Key:      "demo",
		Operator: "Equal",
		Value:    "demo",
		Effect:   "NoSchedule",
	}
	helmValues.Tolerations = []model.Toleration{dummyToleration}
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.Database = "neo4j1"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing helm template on backup helm chart with tolerations")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	tolerations := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Tolerations
	assert.Len(t, tolerations, 1, "more than one tolerations found")
	for _, toleration := range tolerations {
		assert.Equal(t, toleration.Key, dummyToleration.Key, fmt.Sprintf("Toleration key found %s not matching with %s", toleration.Key, dummyToleration.Key))
		assert.Equal(t, string(toleration.Operator), dummyToleration.Operator, fmt.Sprintf("Toleration operator found %s not matching with %s", toleration.Operator, dummyToleration.Operator))
		assert.Equal(t, toleration.Value, dummyToleration.Value, fmt.Sprintf("Toleration value found %s not matching with %s", toleration.Value, dummyToleration.Value))
		assert.Equal(t, string(toleration.Effect), dummyToleration.Effect, fmt.Sprintf("Toleration effect found %s not matching with %s", toleration.Effect, dummyToleration.Effect))
	}
}

// TestNeo4jBackupPodAffinity checks for affinity in the backup cronjob
func TestNeo4jBackupPodAffinity(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues

	helmValues.Affinity = model.Affinity{PodAffinity: model.PodAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []model.RequiredDuringSchedulingIgnoredDuringExecution{
			{
				LabelSelector: model.LabelSelector{
					MatchExpressions: []model.MatchExpressions{
						{
							Key:      "demo",
							Operator: "demo",
							Values:   []string{"demo"},
						},
					},
				},
				TopologyKey: "demo"},
		},
	}}

	helmValues.DisableLookups = true
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.Database = "neo4j1"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing helm template on backup helm chart with affinity")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	affinity := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Affinity
	assert.NotNil(t, affinity.PodAffinity, "affinity missing from backup pod")
	assert.Equal(t, len(affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution), 1)
}

// TestBackupEmptySecretKeyNameWithoutSecretNameAndServiceAccountName checks for error when serviceAccountName and secretName , secretKeyName are missing
func TestBackupEmptySecretKeyNameWithoutSecretNameAndServiceAccountName(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.SecretName = ""
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.Error(t, err, "error must be seen while trying to install helm backup")
	assert.Contains(t, err.Error(), "Please provide either secretName or serviceAccountName. Both cannot be empty.")
}

// TestBackupAzureStorageAccountNameWithSecretNameAndServiceAccountName checks for error when serviceAccountName and secretName , secretKeyName are missing
func TestBackupAzureStorageAccountNameWithSecretNameAndServiceAccountName(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.AzureStorageAccountName = "demo"
	helmValues.ServiceAccountName = "saName"
	helmValues.Backup.CloudProvider = "azure"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.Error(t, err, "error must not be seen while trying to install helm backup")
	assert.Contains(t, err.Error(), "Both secretName|secretKeyName and azureStorageAccountName key cannot be present")
}

// TestBackupAzureStorageAccountNameWithoutSecretNameAndServiceAccountName checks for error when serviceAccountName and secretName , secretKeyName are missing
func TestBackupAzureStorageAccountNameWithoutSecretNameAndServiceAccountName(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.SecretName = ""
	helmValues.Backup.AzureStorageAccountName = ""
	helmValues.ServiceAccountName = ""
	helmValues.Backup.CloudProvider = "azure"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.Error(t, err, "error must not be seen while trying to install helm backup")
	assert.Contains(t, err.Error(), "Both secretName|secretKeyName and azureStorageAccountName key cannot be empty")
}

// TestNeo4jBackupResourcesAndLimits checks for requests and limits (cpu and memory) fields
func TestNeo4jBackupResources(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues

	helmValues.Resources.Requests.CPU = "1"
	helmValues.Resources.Requests.Memory = "2"
	helmValues.Resources.Limits.CPU = "2"
	helmValues.Resources.Limits.Memory = "4"
	helmValues.DisableLookups = true
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.Database = "neo4j1"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing helm template on backup helm chart with affinity")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	resources := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources
	assert.NotNil(t, resources, "resources missing from backup pod")
	assert.Equal(t, resources.Limits.Cpu().String(), helmValues.Resources.Limits.CPU)
	assert.Equal(t, resources.Requests.Cpu().String(), helmValues.Resources.Requests.CPU)
	assert.Equal(t, resources.Limits.Memory().String(), helmValues.Resources.Limits.Memory)
	assert.Equal(t, resources.Requests.Memory().String(), helmValues.Resources.Requests.Memory)
}

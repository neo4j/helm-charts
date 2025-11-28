package unit_tests

import (
	"fmt"
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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

// TestBackupEmptySecretKeyNameWithSecretName checks for empty secretkeyname when secretname is provided
func TestBackupEmptySecretKeyNameWithSecretName(t *testing.T) {
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
	assert.Contains(t, err.Error(), fmt.Sprintf("Secret %s configured in 'backup.secretName' not found", helmValues.Backup.SecretName))
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

// TestBackupNodeSelectorLabels checks nodeSelector labels with disableLookups set to true
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
	helmValues.Backup.AggregateBackup = model.AggregateBackup{}
	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing helm template on backup helm chart with affinity")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	affinity := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Affinity
	assert.NotNil(t, affinity.PodAffinity, "affinity missing from backup pod")
	assert.Equal(t, len(affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution), 1)
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

// TestEmptyBucketName checks for error message when bucketname is not provided
func TestEmptyBucketName(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues

	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = ""
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.Error(t, err, "error not seen while checking for empty bucket name")
	assert.Contains(t, err.Error(), "Empty bucketName. Please set bucketName via --set backup.bucketName")

}

// TestCustomS3EndpointConfiguration tests that custom S3 endpoint configuration is properly set
func TestCustomS3EndpointConfiguration(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "test-bucket"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j"

	// Set custom S3 endpoint configuration
	helmValues.Backup.S3Endpoint = "https://s3.example.com"
	helmValues.Backup.S3ForcePathStyle = true
	helmValues.Backup.S3Region = "us-east-1"
	helmValues.Backup.S3SignatureVersion = "4"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while trying to install helm backup with custom S3 endpoint")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)

	containers := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1, "there should be only one container present")
	container := containers[0]

	// Verify that the custom S3 endpoint environment variables are properly set
	envVariables := container.Env

	// Check for AWS_ENDPOINT_URL_S3
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "AWS_ENDPOINT_URL_S3", Value: "https://s3.example.com"},
		"AWS_ENDPOINT_URL_S3 should be set to custom endpoint")

	// Check for S3_FORCE_PATH_STYLE
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "S3_FORCE_PATH_STYLE", Value: "true"},
		"S3_FORCE_PATH_STYLE should be set to true")

	// Check for AWS_REGION
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "AWS_REGION", Value: "us-east-1"},
		"AWS_REGION should be set to custom region")

	// Check for AWS_DEFAULT_REGION
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "AWS_DEFAULT_REGION", Value: "us-east-1"},
		"AWS_DEFAULT_REGION should be set to custom region")

	// Check for S3_SIGNATURE_VERSION
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "S3_SIGNATURE_VERSION", Value: "4"},
		"S3_SIGNATURE_VERSION should be set to 4")
}

// TestDefaultS3Configuration tests that default S3 configuration works without custom endpoint
func TestDefaultS3Configuration(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "test-bucket"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j"

	// Don't set custom S3 endpoint - use default AWS S3
	// helmValues.Backup.S3Endpoint = "" // This should be empty

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while trying to install helm backup with default S3 configuration")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)

	containers := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1, "there should be only one container present")
	container := containers[0]

	// Verify that AWS_ENDPOINT_URL_S3 is NOT set when no custom endpoint is configured
	envVariables := container.Env

	for _, envVar := range envVariables {
		assert.NotEqual(t, "AWS_ENDPOINT_URL_S3", envVar.Name,
			"AWS_ENDPOINT_URL_S3 should not be set when using default AWS S3")
	}
}

// TestOnPremScenario checks for any errors when backup is performed on onprem
func TestOnPremScenario(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues

	helmValues.Backup.CloudProvider = ""
	helmValues.Backup.BucketName = ""
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing backup on onprem")

}

// TestAggregateEnabledWithServiceAccount checks for any errors when aggregate backup is performed with service account
func TestAggregateEnabledWithServiceAccount(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues

	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = ""
	helmValues.Backup.AggregateBackup.Enabled = true
	helmValues.Backup.AggregateBackup.FromPath = "s3://demo-bucket"
	helmValues.ServiceAccountName = "demo"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing aggregate backup using serviceaccount")
	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	envVariables := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, variable := range envVariables {
		if variable.Name == "AGGREGATE_BACKUP_FROM_PATH" {
			found = true
			assert.Equal(t, variable.Value, helmValues.Backup.AggregateBackup.FromPath)
			break
		}
	}
	assert.Equal(t, found, true)

}

// TestAggregateEnabledWithoutServiceAccount checks for any errors when aggregate backup is performed without service account
func TestAggregateEnabledWithoutServiceAccount(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.AggregateBackup.Enabled = true

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing aggregate backup without using serviceaccount")

}

// TestNeo4jBackupContainerSecurityContext checks for container security context in the backup cronjob
func TestNeo4jBackupContainerSecurityContext(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.Database = "neo4j1"

	helmValues.ContainerSecurityContext = model.ContainerSecurityContext{
		RunAsNonRoot:             true,
		RunAsUser:                7474,
		RunAsGroup:               7474,
		ReadOnlyRootFilesystem:   false,
		AllowPrivilegeEscalation: false,
		Capabilities: model.Capabilities{
			Drop: []string{"ALL"},
		},
	}

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while trying to install helm backup")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	cronjob := cronjobs[0].(*batchv1.CronJob)
	container := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]

	secContext := container.SecurityContext
	assert.NotNil(t, secContext, "container security context should not be nil")

	// Assert all fields requested by customer
	assert.True(t, *secContext.RunAsNonRoot, "RunAsNonRoot should be true")
	assert.Equal(t, int64(7474), *secContext.RunAsUser, "RunAsUser should be 7474")
	assert.Equal(t, int64(7474), *secContext.RunAsGroup, "RunAsGroup should be 7474")
	assert.False(t, *secContext.ReadOnlyRootFilesystem, "ReadOnlyRootFilesystem should be false")
	assert.False(t, *secContext.AllowPrivilegeEscalation, "AllowPrivilegeEscalation should be false")
	assert.Equal(t, []corev1.Capability{"ALL"}, secContext.Capabilities.Drop, "Capabilities.Drop should contain ALL")
}

// TestMultipleBackupEndpointsUnit checks for multiple backup endpoints in the backup cronjob
func TestBackupMultipleEndpoints(t *testing.T) {
	t.Parallel()

	backupEndpoints := "10.3.3.2:6362,10.3.3.3:6362,10.3.3.4:6362"

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.DatabaseBackupEndpoints = backupEndpoints

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error generating helm template with multiple backup endpoints")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)
	assert.Contains(t, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "DATABASE_BACKUP_ENDPOINTS",
		Value: backupEndpoints,
	}, "backup endpoints not set correctly in cronjob")
}

// TestAggregateBackupWithTempDir checks for tempDir in the aggregate backup
func TestAggregateBackupWithTempDir(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = ""
	helmValues.Backup.AggregateBackup = model.AggregateBackup{
		Enabled:  true,
		FromPath: "s3://demo-bucket",
		TempDir:  "/custom/temp/dir",
	}
	helmValues.ServiceAccountName = "demo"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing aggregate backup with tempDir")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	envVariables := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, variable := range envVariables {
		if variable.Name == "AGGREGATE_BACKUP_TEMP_DIR" {
			found = true
			assert.Equal(t, variable.Value, helmValues.Backup.AggregateBackup.TempDir)
			break
		}
	}
	assert.Equal(t, found, true)
}

// TestAggregateBackupDefaultTempDir checks that aggregate backup uses /backups as default temp directory
func TestAggregateBackupDefaultTempDir(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "test-bucket"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.AggregateBackup = model.AggregateBackup{
		Enabled:  true,
		FromPath: "s3://demo-bucket",
		// No TempDir specified - should default to /backups
	}
	helmValues.ServiceAccountName = "demo"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing aggregate backup with default tempDir")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	envVariables := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env

	// AGGREGATE_BACKUP_TEMP_DIR should not be set when using default
	for _, variable := range envVariables {
		if variable.Name == "AGGREGATE_BACKUP_TEMP_DIR" {
			assert.Equal(t, "", variable.Value, "AGGREGATE_BACKUP_TEMP_DIR should be empty when using default")
		}
	}
}

// TestBackupS3CASecretValidation checks that s3CASecretKey is required when s3CASecretName is provided
func TestBackupS3CASecretValidation(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "demo1"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.S3CASecretName = "my-ca-cert"

	_, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "If backup.s3CASecretName is specified, backup.s3CASecretKey must also be specified")
}

// TestBackupS3CASecretConfiguration checks the S3 CA certificate configuration
func TestBackupS3CASecretConfiguration(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "demo1"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.S3CASecretName = "my-ca-cert"
	helmValues.Backup.S3CASecretKey = "ca.crt"
	helmValues.Backup.S3Endpoint = "s3.example.com"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)
	cronjob := cronjobs[0].(*batchv1.CronJob)

	// TLS is now automatically handled by AWS_ENDPOINT_URL_S3 (https:// = TLS enabled)

	var certMountFound bool
	for _, volumeMount := range cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts {
		if volumeMount.Name == "s3-ca-cert" {
			certMountFound = true
			assert.Equal(t, "/s3-ca-cert", volumeMount.MountPath)
			assert.True(t, volumeMount.ReadOnly)
		}
	}
	assert.True(t, certMountFound, "s3-ca-cert volume mount not found")

	var certVolumeFound bool
	for _, volume := range cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes {
		if volume.Name == "s3-ca-cert" {
			certVolumeFound = true
			assert.Equal(t, "my-ca-cert", volume.Secret.SecretName)
			assert.Len(t, volume.Secret.Items, 1)
			assert.Equal(t, "ca.crt", volume.Secret.Items[0].Key)
			assert.Equal(t, "ca.crt", volume.Secret.Items[0].Path)
		}
	}
	assert.True(t, certVolumeFound, "s3-ca-cert volume not found")
}

// TestBackupS3GenericParameters checks that the new S3 parameters are correctly set in the environment variables
func TestBackupS3GenericParameters(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup = model.Backup{
		BucketName:               "test-bucket",
		DatabaseAdminServiceName: "neo4j-admin",
		CloudProvider:            "aws",
		SecretName:               "demo",
		SecretKeyName:            "credentials",
		S3Endpoint:               "https://s3.example.com",
		S3ForcePathStyle:         true,
		S3Region:                 "us-east-1",
		S3SignatureVersion:       "4",
	}

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	envVariables := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env

	// Check that the S3 parameters are correctly set in the environment variables
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "S3_FORCE_PATH_STYLE", Value: "true"})
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "AWS_REGION", Value: "us-east-1"})
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "S3_SIGNATURE_VERSION", Value: "4"})
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "AWS_ENDPOINT_URL_S3", Value: "https://s3.example.com"})
}

// TestBackupCompressEnvVarDefaultTrue checks that the Compress value is set to true correctly when the variable is not specified
func TestBackupCompressEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()
	// Test without setting helmValues.Backup.Compress
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.Compress = true

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "COMPRESS" {
			found = true
			assert.Equal(t, "true", env.Value, "Expected COMPRESS to be true by default")
			break
		}
	}
	assert.True(t, found, "COMPRESS env var not found")
}

// TestBackupCompressEnvVarFalse checks that the Compress value is set to false when explicitly set as such
func TestBackupCompressEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.Compress = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "COMPRESS" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected COMPRESS to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "COMPRESS env var not found")
}

// TestBackupWithTempDir checks for tempDir in the regular backup
func TestBackupWithTempDir(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "test-bucket"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.TempDir = "/custom/backup/temp/dir"
	helmValues.ServiceAccountName = "demo"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing backup with tempDir")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	envVariables := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, variable := range envVariables {
		if variable.Name == "BACKUP_TEMP_DIR" {
			found = true
			assert.Equal(t, variable.Value, helmValues.Backup.TempDir)
			break
		}
	}
	assert.Equal(t, found, true)
}

// TestBackupVerboseEnvVarDefaultTrue checks that the Verbose value is set to true correctly
func TestBackupVerboseEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "test-bucket"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Verbose = true
	helmValues.ServiceAccountName = "demo"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing backup with verbose")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	envVariables := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, variable := range envVariables {
		if variable.Name == "VERBOSE" {
			found = true
			assert.Equal(t, "true", variable.Value)
			break
		}
	}
	assert.True(t, found, "VERBOSE env var not found")
}

// TestBackupVerboseEnvVarFalse checks that the Verbose value is set to false when explicitly set as such
func TestBackupVerboseEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.Verbose = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "VERBOSE" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected VERBOSE to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "VERBOSE env var not found")
}

// TestBackupRemoteAddressResolutionEnvVarTrue checks that the RemoteAddressResolution value is set to true correctly
func TestBackupRemoteAddressResolutionEnvVarTrue(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "test-bucket"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.RemoteAddressResolution = true
	helmValues.ServiceAccountName = "demo"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing backup with remoteAddressResolution")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	envVariables := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, variable := range envVariables {
		if variable.Name == "REMOTE_ADDRESS_RESOLUTION" {
			found = true
			assert.Equal(t, "true", variable.Value, "Expected REMOTE_ADDRESS_RESOLUTION to be true when explicitly enabled")
			break
		}
	}
	assert.True(t, found, "REMOTE_ADDRESS_RESOLUTION env var not found")
}

// TestBackupRemoteAddressResolutionEnvVarFalse checks that the RemoteAddressResolution value is set to false when explicitly set as such
func TestBackupRemoteAddressResolutionEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.RemoteAddressResolution = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "REMOTE_ADDRESS_RESOLUTION" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected REMOTE_ADDRESS_RESOLUTION to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "REMOTE_ADDRESS_RESOLUTION env var not found")
}

// TestBackupRemoteAddressResolutionEnvVarDefaultFalse checks that the RemoteAddressResolution value defaults to false
func TestBackupRemoteAddressResolutionEnvVarDefaultFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	// RemoteAddressResolution is not explicitly set, should default to false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "REMOTE_ADDRESS_RESOLUTION" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected REMOTE_ADDRESS_RESOLUTION to default to false")
			break
		}
	}
	assert.True(t, found, "REMOTE_ADDRESS_RESOLUTION env var not found")
}

// TestBackupKeepBackupFilesEnvVarDefaultTrue checks that the KeepBackupFiles value is set to true correctly
func TestBackupKeepBackupFilesEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.KeepBackupFiles = true

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "KEEP_BACKUP_FILES" {
			found = true
			assert.Equal(t, "true", env.Value, "Expected KEEP_BACKUP_FILES to be true by default")
			break
		}
	}
	assert.True(t, found, "KEEP_BACKUP_FILES env var not found")
}

// TestBackupKeepBackupFilesEnvVarFalse checks that the KeepBackupFiles value is set to false when explicitly set as such
func TestBackupKeepBackupFilesEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.KeepBackupFiles = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "KEEP_BACKUP_FILES" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected KEEP_BACKUP_FILES to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "KEEP_BACKUP_FILES env var not found")
}

// TestS3ForcePathStyleEnvVarDefaultTrue checks that the S3ForcePathStyle value is set to true correctly
func TestS3ForcePathStyleEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.S3ForcePathStyle = true

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "S3_FORCE_PATH_STYLE" {
			found = true
			assert.Equal(t, "true", env.Value, "Expected S3_FORCE_PATH_STYLE to be true by default")
			break
		}
	}
	assert.True(t, found, "S3_FORCE_PATH_STYLE env var not found")
}

// TestS3ForcePathStyleEnvVarFalse checks that the S3ForcePathStyle value is set to false when explicitly set as such
func TestS3ForcePathStyleEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.S3ForcePathStyle = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "S3_FORCE_PATH_STYLE" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected S3_FORCE_PATH_STYLE to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "S3_FORCE_PATH_STYLE env var not found")
}

// TestAggregateBackupVerboseEnvVarDefaultTrue checks that the AggregateBackup.Verbose value is set to true correctly
func TestAggregateBackupVerboseEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.AggregateBackup.Enabled = true
	helmValues.Backup.AggregateBackup.Verbose = true

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "AGGREGATE_BACKUP_VERBOSE" {
			found = true
			assert.Equal(t, "true", env.Value, "Expected AGGREGATE_BACKUP_VERBOSE to be true by default")
			break
		}
	}
	assert.True(t, found, "AGGREGATE_BACKUP_VERBOSE env var not found")
}

// TestAggregateBackupVerboseEnvVarFalse checks that the AggregateBackup.Verbose value is set to false when explicitly set as such
func TestAggregateBackupVerboseEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.AggregateBackup.Enabled = true
	helmValues.Backup.AggregateBackup.Verbose = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "AGGREGATE_BACKUP_VERBOSE" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected AGGREGATE_BACKUP_VERBOSE to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "AGGREGATE_BACKUP_VERBOSE env var not found")
}

// TestConsistencyCheckCheckIndexesEnvVarDefaultTrue checks that the ConsistencyCheck.CheckIndexes value is set to true correctly
func TestConsistencyCheckCheckIndexesEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.CheckIndexes = true

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_INDEXES" {
			found = true
			assert.Equal(t, "true", env.Value, "Expected CONSISTENCY_CHECK_INDEXES to be true by default")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_INDEXES env var not found")
}

// TestConsistencyCheckCheckIndexesEnvVarFalse checks that the ConsistencyCheck.CheckIndexes value is set to false when explicitly set as such
func TestConsistencyCheckCheckIndexesEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.CheckIndexes = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_INDEXES" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected CONSISTENCY_CHECK_INDEXES to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_INDEXES env var not found")
}

// TestConsistencyCheckCheckGraphEnvVarDefaultTrue checks that the ConsistencyCheck.CheckGraph value is set to true correctly
func TestConsistencyCheckCheckGraphEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.CheckGraph = true

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_GRAPH" {
			found = true
			assert.Equal(t, "true", env.Value, "Expected CONSISTENCY_CHECK_GRAPH to be true by default")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_GRAPH env var not found")
}

// TestConsistencyCheckCheckGraphEnvVarFalse checks that the ConsistencyCheck.CheckGraph value is set to false when explicitly set as such
func TestConsistencyCheckCheckGraphEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.CheckGraph = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_GRAPH" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected CONSISTENCY_CHECK_GRAPH to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_GRAPH env var not found")
}

// TestConsistencyCheckCheckCountsEnvVarDefaultTrue checks that the ConsistencyCheck.CheckCounts value is set to true correctly
func TestConsistencyCheckCheckCountsEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.CheckCounts = true

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_COUNTS" {
			found = true
			assert.Equal(t, "true", env.Value, "Expected CONSISTENCY_CHECK_COUNTS to be true by default")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_COUNTS env var not found")
}

// TestConsistencyCheckCheckCountsEnvVarFalse checks that the ConsistencyCheck.CheckCounts value is set to false when explicitly set as such
func TestConsistencyCheckCheckCountsEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.CheckCounts = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_COUNTS" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected CONSISTENCY_CHECK_COUNTS to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_COUNTS env var not found")
}

// TestConsistencyCheckCheckPropertyOwnersEnvVarDefaultTrue checks that the ConsistencyCheck.CheckPropertyOwners value is set to true correctly
func TestConsistencyCheckCheckPropertyOwnersEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.CheckPropertyOwners = true

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_PROPERTYOWNERS" {
			found = true
			assert.Equal(t, "true", env.Value, "Expected CONSISTENCY_CHECK_PROPERTYOWNERS to be true by default")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_PROPERTYOWNERS env var not found")
}

// TestConsistencyCheckCheckPropertyOwnersEnvVarFalse checks that the ConsistencyCheck.CheckPropertyOwners value is set to false when explicitly set as such
func TestConsistencyCheckCheckPropertyOwnersEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.CheckPropertyOwners = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_PROPERTYOWNERS" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected CONSISTENCY_CHECK_PROPERTYOWNERS to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_PROPERTYOWNERS env var not found")
}

// TestConsistencyCheckVerboseEnvVarDefaultTrue checks that the ConsistencyCheck.Verbose value is set to true correctly
func TestConsistencyCheckVerboseEnvVarDefaultTrue(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.Verbose = true

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_VERBOSE" {
			found = true
			assert.Equal(t, "true", env.Value, "Expected CONSISTENCY_CHECK_VERBOSE to be true by default")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_VERBOSE env var not found")
}

// TestConsistencyCheckVerboseEnvVarFalse checks that the ConsistencyCheck.Verbose value is set to false when explicitly set as such
func TestConsistencyCheckVerboseEnvVarFalse(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.ConsistencyCheck.Verbose = false

	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "key"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_VERBOSE" {
			found = true
			assert.Equal(t, "false", env.Value, "Expected CONSISTENCY_CHECK_VERBOSE to be false when explicitly disabled")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_VERBOSE env var not found")
}

// TestBackupWithRegistryAndPullSecrets checks if registry is used in image and pull secrets are set
func TestBackupWithRegistryAndPullSecrets(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Neo4J.Registry = "myregistry.com"
	helmValues.Neo4J.ImagePullSecrets = []string{"my-pull-secret"}

	// Set required fields
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "credentials"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "test-bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	cronjob := cronjobs[0].(*batchv1.CronJob)
	container := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]

	expectedImage := "myregistry.com/" + helmValues.Neo4J.Image + ":" + helmValues.Neo4J.ImageTag
	assert.Equal(t, expectedImage, container.Image)

	pullSecrets := cronjob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets
	assert.Len(t, pullSecrets, 1)
	assert.Equal(t, "my-pull-secret", pullSecrets[0].Name)
}

// TestConsistencyCheckTimeoutDefaultValue checks that the default timeout is set correctly for cloud storage
func TestConsistencyCheckTimeoutDefaultValue(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "credentials"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "test-bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"
	helmValues.ConsistencyCheck.Enable = true
	// timeout not specified - should default to "30m" for cloud storage

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_TIMEOUT" {
			found = true
			assert.Equal(t, "30m", env.Value, "Expected CONSISTENCY_CHECK_TIMEOUT to default to 30m for cloud storage")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_TIMEOUT env var not found")
}

// TestConsistencyCheckTimeoutCustomValue checks that a custom timeout value is set correctly
func TestConsistencyCheckTimeoutCustomValue(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.SecretKeyName = "credentials"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "test-bucket"
	helmValues.Backup.DatabaseAdminServiceName = "admin"
	helmValues.Backup.Database = "neo4j"
	helmValues.ConsistencyCheck.Enable = true
	helmValues.ConsistencyCheck.Timeout = "2h"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1)

	envVars := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, env := range envVars {
		if env.Name == "CONSISTENCY_CHECK_TIMEOUT" {
			found = true
			assert.Equal(t, "2h", env.Value, "Expected CONSISTENCY_CHECK_TIMEOUT to be set to custom value 2h")
			break
		}
	}
	assert.True(t, found, "CONSISTENCY_CHECK_TIMEOUT env var not found")
}

// TestAzureBlobServiceURLConfiguration tests that Azure blob service URL configuration is properly set
func TestAzureBlobServiceURLConfiguration(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.AzureStorageAccountName = "testaccount"
	helmValues.Backup.CloudProvider = "azure"
	helmValues.Backup.BucketName = "test-container"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j"
	helmValues.ServiceAccountName = "azure-sa"

	// Set custom Azure blob service URL for Azure Government Cloud
	helmValues.Backup.AzureBlobServiceURL = "blob.core.usgovcloudapi.net"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while trying to install helm backup with Azure blob service URL")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)

	containers := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1, "there should be only one container present")
	container := containers[0]

	// Verify that the Azure blob service URL environment variables are properly set
	envVariables := container.Env

	// Check for AZURE_BLOB_SERVICE_URL (used by Go SDK client)
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "AZURE_BLOB_SERVICE_URL", Value: "blob.core.usgovcloudapi.net"},
		"AZURE_BLOB_SERVICE_URL should be set to custom endpoint")

	// Check for NEO4J_dbms_integrations_cloud__storage_azb_blob__endpoint__suffix (used by Neo4j native backup)
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "NEO4J_dbms_integrations_cloud__storage_azb_blob__endpoint__suffix", Value: "blob.core.usgovcloudapi.net"},
		"NEO4J_dbms_integrations_cloud__storage_azb_blob__endpoint__suffix should be set to custom endpoint")
}

// TestAzureBlobServiceURLDefaultConfiguration tests that default Azure blob service URL works without custom endpoint
func TestAzureBlobServiceURLDefaultConfiguration(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.AzureStorageAccountName = "testaccount"
	helmValues.Backup.CloudProvider = "azure"
	helmValues.Backup.BucketName = "test-container"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j"
	helmValues.ServiceAccountName = "azure-sa"

	// Don't set custom Azure blob service URL - should use default

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while trying to install helm backup with default Azure configuration")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")
	cronjob := cronjobs[0].(*batchv1.CronJob)

	containers := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1, "there should be only one container present")
	container := containers[0]

	// Verify that AZURE_BLOB_SERVICE_URL is set to empty string (will use default)
	envVariables := container.Env

	// AZURE_BLOB_SERVICE_URL should be present but empty
	for _, envVar := range envVariables {
		if envVar.Name == "AZURE_BLOB_SERVICE_URL" {
			assert.Equal(t, "", envVar.Value, "AZURE_BLOB_SERVICE_URL should be empty when using default")
		}
		// NEO4J_dbms_integrations_cloud__storage_azb_blob__endpoint__suffix should NOT be present when no custom URL is set
		assert.NotEqual(t, "NEO4J_dbms_integrations_cloud__storage_azb_blob__endpoint__suffix", envVar.Name,
			"NEO4J_dbms_integrations_cloud__storage_azb_blob__endpoint__suffix should not be set when using default")
	}
}

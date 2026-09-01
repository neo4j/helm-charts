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

	// Check that TLS is automatically enabled
	var tlsEnabled bool
	for _, env := range cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "S3_ENDPOINT_TLS" {
			tlsEnabled = env.Value == "true"
		}
	}
	assert.True(t, tlsEnabled, "S3_ENDPOINT_TLS should be true when s3CASecretName is provided")

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
		S3EndpointTLS:            true,
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
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "S3_REGION", Value: "us-east-1"})
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "S3_SIGNATURE_VERSION", Value: "4"})
}

// TestAggregateBackupS3ChecksumConfiguration checks that S3 checksum options are set for aggregate backups
func TestAggregateBackupS3ChecksumConfiguration(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.AggregateBackup = model.AggregateBackup{
		Enabled:  true,
		FromPath: "s3://test-bucket",
	}
	helmValues.Backup.S3RequestChecksumCalculation = "WHEN_REQUIRED"
	helmValues.Backup.S3ResponseChecksumValidation = "WHEN_REQUIRED"
	helmValues.Backup.S3DisableMultipartChecksums = true
	helmValues.ServiceAccountName = "backup-service-account"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	envVariables := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "AWS_REQUEST_CHECKSUM_CALCULATION", Value: "WHEN_REQUIRED"})
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "AWS_RESPONSE_CHECKSUM_VALIDATION", Value: "WHEN_REQUIRED"})
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "AWS_S3_DISABLE_MULTIPART_CHECKSUMS", Value: "true"})
}

// TestAggregateBackupS3ChecksumDefaults checks that unset S3 checksum options preserve AWS SDK defaults
func TestAggregateBackupS3ChecksumDefaults(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.AggregateBackup = model.AggregateBackup{
		Enabled:  true,
		FromPath: "s3://test-bucket",
	}
	helmValues.ServiceAccountName = "backup-service-account"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err)

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	checksumEnvNames := []string{
		"AWS_REQUEST_CHECKSUM_CALCULATION",
		"AWS_RESPONSE_CHECKSUM_VALIDATION",
		"AWS_S3_DISABLE_MULTIPART_CHECKSUMS",
	}
	envVariables := cronjobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	for _, envVariable := range envVariables {
		assert.NotContains(t, checksumEnvNames, envVariable.Name)
	}
}

// TestBackupAzureBlobServiceURL checks that the Azure blob service URL is properly configured
func TestBackupAzureBlobServiceURL(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.CloudProvider = "azure"
	helmValues.Backup.BucketName = "test-container"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j"
	helmValues.Backup.AzureStorageAccountName = "testgovaccount"
	helmValues.Backup.AzureBlobServiceURL = "blob.core.usgovcloudapi.net"
	helmValues.ServiceAccountName = "azure-backup-sa"

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing helm template on backup helm chart with Azure blob service URL")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	cronjob := cronjobs[0].(*batchv1.CronJob)
	container := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]

	// Check that AZURE_BLOB_SERVICE_URL environment variable is set
	var azureBlobServiceURLEnv *corev1.EnvVar
	for _, env := range container.Env {
		if env.Name == "AZURE_BLOB_SERVICE_URL" {
			azureBlobServiceURLEnv = &env
			break
		}
	}

	assert.NotNil(t, azureBlobServiceURLEnv, "AZURE_BLOB_SERVICE_URL environment variable should be present")
	assert.Equal(t, "blob.core.usgovcloudapi.net", azureBlobServiceURLEnv.Value, "AZURE_BLOB_SERVICE_URL should match the configured value")
}

// TestBackupAzureBlobServiceURLDefault checks that the Azure blob service URL defaults to empty when not specified
func TestBackupAzureBlobServiceURLDefault(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.DisableLookups = true
	helmValues.Backup.CloudProvider = "azure"
	helmValues.Backup.BucketName = "test-container"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j"
	helmValues.Backup.AzureStorageAccountName = "testaccount"
	helmValues.ServiceAccountName = "azure-backup-sa"
	// Not setting AzureBlobServiceURL to test default behavior

	manifests, err := model.HelmTemplateFromStruct(t, model.BackupHelmChart, helmValues)
	assert.NoError(t, err, "error seen while performing helm template on backup helm chart without Azure blob service URL")

	cronjobs := manifests.OfType(&batchv1.CronJob{})
	assert.Len(t, cronjobs, 1, "there should be only one cronjob")

	cronjob := cronjobs[0].(*batchv1.CronJob)
	container := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]

	// Check that AZURE_BLOB_SERVICE_URL environment variable is set but empty
	var azureBlobServiceURLEnv *corev1.EnvVar
	for _, env := range container.Env {
		if env.Name == "AZURE_BLOB_SERVICE_URL" {
			azureBlobServiceURLEnv = &env
			break
		}
	}

	assert.NotNil(t, azureBlobServiceURLEnv, "AZURE_BLOB_SERVICE_URL environment variable should be present")
	assert.Equal(t, "", azureBlobServiceURLEnv.Value, "AZURE_BLOB_SERVICE_URL should be empty when not configured")
}

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

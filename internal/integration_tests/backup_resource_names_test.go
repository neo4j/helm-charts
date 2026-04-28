package integration_tests

import (
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestBackupSecretNameUsesReleaseName(t *testing.T) {
	firstRelease := model.NewReleaseName("cluster-backup-aws-s3-test-run")
	secondRelease := model.NewReleaseName("cluster-backup-aws-s3-tls-test-run")

	firstSecret := backupSecretName(firstRelease, "awscred")
	secondSecret := backupSecretName(secondRelease, "awscred")

	assert.Equal(t, "cluster-backup-aws-s3-test-run-awscred", firstSecret)
	assert.Equal(t, "cluster-backup-aws-s3-tls-test-run-awscred", secondSecret)
	assert.NotEqual(t, firstSecret, secondSecret)
}

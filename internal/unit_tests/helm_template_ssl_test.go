package unit_tests

import (
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

func TestSSLSecretsAreProjectedWithoutSubPath(t *testing.T) {
	t.Parallel()

	args := append([]string{}, useDataModeAndAcceptLicense...)
	args = append(args, useNeo4jClusterName...)
	args = append(args,
		"--set", "ssl.bolt.privateKey.secretName=bolt-key-secret",
		"--set", "ssl.bolt.privateKey.subPath=tls.key",
		"--set", "ssl.bolt.publicCertificate.secretName=bolt-cert-secret",
		"--set", "ssl.bolt.publicCertificate.subPath=tls.crt",
	)

	manifest, err := model.HelmTemplate(t, model.HelmChart, args)
	require.NoError(t, err)

	statefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
	neo4jContainer := statefulSet.Spec.Template.Spec.Containers[0]

	var certificateMount *v1.VolumeMount
	for i := range neo4jContainer.VolumeMounts {
		if neo4jContainer.VolumeMounts[i].Name == "bolt-certificates" {
			certificateMount = &neo4jContainer.VolumeMounts[i]
			break
		}
	}
	require.NotNil(t, certificateMount)
	assert.Equal(t, "/var/lib/neo4j/certificates/bolt", certificateMount.MountPath)
	assert.Empty(t, certificateMount.SubPath)
	assert.Empty(t, certificateMount.SubPathExpr)
	assert.True(t, certificateMount.ReadOnly)

	var certificateVolume *v1.Volume
	for i := range statefulSet.Spec.Template.Spec.Volumes {
		if statefulSet.Spec.Template.Spec.Volumes[i].Name == "bolt-certificates" {
			certificateVolume = &statefulSet.Spec.Template.Spec.Volumes[i]
			break
		}
	}
	require.NotNil(t, certificateVolume)
	require.NotNil(t, certificateVolume.Projected)
	require.Len(t, certificateVolume.Projected.Sources, 2)

	certificateSecret := certificateVolume.Projected.Sources[0].Secret
	require.NotNil(t, certificateSecret)
	assert.Equal(t, "bolt-cert-secret", certificateSecret.Name)
	assert.Equal(t, []v1.KeyToPath{{Key: "tls.crt", Path: "public.crt"}}, certificateSecret.Items)

	privateKeySecret := certificateVolume.Projected.Sources[1].Secret
	require.NotNil(t, privateKeySecret)
	assert.Equal(t, "bolt-key-secret", privateKeySecret.Name)
	assert.Equal(t, []v1.KeyToPath{{Key: "tls.key", Path: "private.key"}}, privateKeySecret.Items)
}

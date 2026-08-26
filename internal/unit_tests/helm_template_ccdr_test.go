package unit_tests

import (
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestCCDRServiceDisabledByDefault(t *testing.T) {
	t.Parallel()

	manifest, err := model.HelmTemplate(t, model.HelmChart, append(useEnterpriseAndAcceptLicense, requiredDataMode...), append(useNeo4jClusterName, enableCluster...)...)
	if !assert.NoError(t, err) {
		return
	}

	assert.Nil(t, manifest.OfTypeWithName(&v1.Service{}, model.DefaultHelmTemplateReleaseName.String()+"-ccdr"))
}

func TestClusterCustomServiceAccountReceivesDiscoveryRBAC(t *testing.T) {
	t.Parallel()

	args := append([]string{}, useEnterpriseAndAcceptLicense...)
	args = append(args, requiredDataMode...)
	args = append(args, useNeo4jClusterName...)
	args = append(args, enableCluster...)
	args = append(args, "--set-string", "podSpec.serviceAccountName=neo4j-backup-reader")

	manifest, err := model.HelmTemplate(t, model.HelmChart, args)
	if !assert.NoError(t, err) {
		return
	}

	statefulSet := manifest.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String()).(*appsv1.StatefulSet)
	assert.Equal(t, "neo4j-backup-reader", statefulSet.Spec.Template.Spec.ServiceAccountName)

	roleBindingName := model.DefaultHelmTemplateReleaseName.String() + "-service-binding"
	roleBinding := manifest.OfTypeWithName(&rbacv1.RoleBinding{}, roleBindingName).(*rbacv1.RoleBinding)
	if assert.Len(t, roleBinding.Subjects, 1) {
		assert.Equal(t, "neo4j-backup-reader", roleBinding.Subjects[0].Name)
	}
	assert.Equal(t, model.DefaultHelmTemplateReleaseName.String()+"-service-reader", roleBinding.RoleRef.Name)
	assert.Nil(t, manifest.OfTypeWithName(&v1.ServiceAccount{}, model.DefaultHelmTemplateReleaseName.String()))
}

func TestCCDRServiceExposesOnlyClusterPortForOneMember(t *testing.T) {
	t.Parallel()

	args := append([]string{}, useEnterpriseAndAcceptLicense...)
	args = append(args, requiredDataMode...)
	args = append(args, useNeo4jClusterName...)
	args = append(args, enableCluster...)
	args = append(args,
		"--set", "services.ccdr.enabled=true",
		"--set", "services.ccdr.spec.type=LoadBalancer",
		"--set", "services.ccdr.spec.externalTrafficPolicy=Local",
		"--set", "services.ccdr.spec.loadBalancerSourceRanges[0]=10.0.0.0/8",
		"--set-string", "services.ccdr.annotations.example\\.com/ccdr=enabled",
	)

	manifest, err := model.HelmTemplate(t, model.HelmChart, args)
	if !assert.NoError(t, err) {
		return
	}

	service := manifest.OfTypeWithName(&v1.Service{}, model.DefaultHelmTemplateReleaseName.String()+"-ccdr").(*v1.Service)
	assert.Equal(t, v1.ServiceTypeLoadBalancer, service.Spec.Type)
	assert.True(t, service.Spec.PublishNotReadyAddresses)
	assert.Equal(t, v1.ServiceExternalTrafficPolicyLocal, service.Spec.ExternalTrafficPolicy)
	assert.Equal(t, []string{"10.0.0.0/8"}, service.Spec.LoadBalancerSourceRanges)
	assert.Equal(t, "enabled", service.Annotations["example.com/ccdr"])
	assert.Equal(t, map[string]string{
		"app":                     "neo4j-cluster",
		"helm.neo4j.com/instance": model.DefaultHelmTemplateReleaseName.String(),
	}, service.Spec.Selector)
	if assert.Len(t, service.Spec.Ports, 1) {
		port := service.Spec.Ports[0]
		assert.Equal(t, "tcp-ccdr", port.Name)
		assert.Equal(t, v1.ProtocolTCP, port.Protocol)
		assert.Equal(t, int32(6000), port.Port)
		assert.Equal(t, int32(6000), port.TargetPort.IntVal)
	}
}

func TestCCDRServiceRequiresEnterpriseCluster(t *testing.T) {
	t.Parallel()

	t.Run("community", func(t *testing.T) {
		args := append([]string{}, useCommunity...)
		args = append(args, requiredDataMode...)
		args = append(args, useNeo4jStandaloneName...)
		args = append(args, "--set", "services.ccdr.enabled=true")
		_, err := model.HelmTemplate(t, model.HelmChart, args)
		assert.ErrorContains(t, err, "services.ccdr.enabled requires neo4j.edition=enterprise")
	})

	t.Run("standalone", func(t *testing.T) {
		args := append([]string{}, useEnterpriseAndAcceptLicense...)
		args = append(args, requiredDataMode...)
		args = append(args, useNeo4jStandaloneName...)
		args = append(args, "--set", "services.ccdr.enabled=true")
		_, err := model.HelmTemplate(t, model.HelmChart, args)
		assert.ErrorContains(t, err, "services.ccdr.enabled requires a Neo4j cluster")
	})
}

func TestClusterTLSValuesRenderFromTypedModel(t *testing.T) {
	t.Parallel()

	values := model.DefaultEnterpriseValues
	values.Neo4J.MinimumClusterSize = 3
	values.Services.CCDR = model.CCDR{Enabled: true, Spec: model.Spec{Type: "ClusterIP"}}
	values.Ssl.Cluster.PrivateKey.SecretName = "cluster-key"
	values.Ssl.Cluster.PublicCertificate.SecretName = "cluster-cert"
	values.Ssl.Cluster.TrustedCerts.Sources = []interface{}{map[string]interface{}{
		"secret": map[string]interface{}{"name": "remote-cluster-ca"},
	}}

	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, values)
	if !assert.NoError(t, err) {
		return
	}

	assert.NotNil(t, manifest.OfTypeWithName(&v1.Service{}, model.DefaultHelmTemplateReleaseName.String()+"-ccdr"))
	statefulSet := manifest.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String()).(*appsv1.StatefulSet)

	volumeNames := map[string]bool{}
	for _, volume := range statefulSet.Spec.Template.Spec.Volumes {
		volumeNames[volume.Name] = true
	}
	assert.True(t, volumeNames["cluster-certificates"])
	assert.True(t, volumeNames["cluster-trusted"])

	mountPaths := map[string]bool{}
	for _, mount := range statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
		mountPaths[mount.MountPath] = true
	}
	assert.True(t, mountPaths["/var/lib/neo4j/certificates/cluster"])
	assert.True(t, mountPaths["/var/lib/neo4j/certificates/cluster/trusted"])
}

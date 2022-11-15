package integration_tests

import (
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"testing"
)

type parallelResult struct {
	Closeable
	error
}

type helmComponent interface {
	Name() model.ReleaseName
	Install(t *testing.T) parallelResult
}

type clusterCore struct {
	name                 model.ReleaseName
	extraHelmInstallArgs []string
}

func (c clusterCore) Name() model.ReleaseName {
	return c.name
}

func (c clusterCore) Install(t *testing.T) parallelResult {
	var err error
	var cleanup Closeable
	cleanup, err = InstallNeo4jInGcloud(t, gcloud.CurrentZone(), gcloud.CurrentProject(), c.name, model.ClusterCoreHelmChart, c.extraHelmInstallArgs...)
	return parallelResult{cleanup, err}
}

type clusterReadReplica struct {
	name                 model.ReleaseName
	extraHelmInstallArgs []string
}

func (c clusterReadReplica) Name() model.ReleaseName {
	return c.name
}

func (c clusterReadReplica) Install(t *testing.T) parallelResult {
	var err error
	var cleanup Closeable
	cleanup, err = InstallNeo4jInGcloud(t, gcloud.CurrentZone(), gcloud.CurrentProject(), c.name, model.ClusterReadReplicaHelmChart, c.extraHelmInstallArgs...)
	return parallelResult{cleanup, err}
}

type clusterLoadBalancer struct {
	name                 model.ReleaseName
	extraHelmInstallArgs []string
}

func (c clusterLoadBalancer) Name() model.ReleaseName {
	return c.name
}

func (c clusterLoadBalancer) Install(t *testing.T) parallelResult {
	var err error
	var cleanup Closeable
	cleanup = func() error {
		return run(t, "helm", model.LoadBalancerHelmCommand("uninstall", c.name, c.extraHelmInstallArgs...)...)
	}
	err = run(t, "helm", model.LoadBalancerHelmCommand("install", c.name, c.extraHelmInstallArgs...)...)
	return parallelResult{cleanup, err}
}

type clusterHeadLessService struct {
	name                 model.ReleaseName
	extraHelmInstallArgs []string
}

func (c clusterHeadLessService) Name() model.ReleaseName {
	return c.name
}

func (c clusterHeadLessService) Install(t *testing.T) parallelResult {
	var err error
	var cleanup Closeable
	cleanup = func() error {
		return run(t, "helm", model.HeadlessServiceHelmCommand("uninstall", c.name, c.extraHelmInstallArgs...)...)
	}
	err = run(t, "helm", model.HeadlessServiceHelmCommand("install", c.name, c.extraHelmInstallArgs...)...)
	return parallelResult{cleanup, err}
}

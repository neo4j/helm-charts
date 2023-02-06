package integration_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
	"testing"
)

func ResourcesCleanup(t *testing.T, releaseName model.ReleaseName) error {
	return run(t, "helm", "uninstall", releaseName.String(), "--namespace", string(releaseName.Namespace()), "--wait", "--timeout=3m")
}

func ResourcesReinstall(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChartBuilder) error {

	err := run(t, "helm", model.BaseHelmCommand("install", releaseName, chart, model.Neo4jEdition, "--wait", "--timeout", "300s")...)
	if err != nil {
		t.Log("Helm Install failed:", err)
		_ = run(t, "kubectl", "get", "events")
		return err
	}
	err = run(t, "kubectl", "--namespace", string(releaseName.Namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/"+releaseName.String())
	if err != nil {
		t.Log("Helm Install failed:", err)
		return err
	}
	return err
}

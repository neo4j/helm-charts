package integration_tests

import (
	"fmt"
	"time"

	"github.com/neo4j/helm-charts/internal/model"
	"testing"
)

func ResourcesCleanup(t *testing.T, releaseName model.ReleaseName) error {
	return run(t, "helm", "uninstall", releaseName.String(), "--namespace", string(releaseName.Namespace()), "--wait", "--timeout=3m")
}

func ResourcesReinstall(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChartBuilder) error {
	namespace := string(releaseName.Namespace())
	// Clean up the load balancer service and endpoint that might be left behind
	// due to helm.sh/resource-policy: keep annotation when clustering is enabled
	loadBalancerServiceName := fmt.Sprintf("%s-lb-neo4j", model.DefaultNeo4jName)
	
	// Delete the service if it exists (it might have resource-policy: keep)
	_ = run(t, "kubectl", "delete", "service", loadBalancerServiceName, "--namespace", namespace, "--ignore-not-found=true")
	
	// Delete the endpoint if it exists (Kubernetes creates this automatically for services)
	_ = run(t, "kubectl", "delete", "endpoints", loadBalancerServiceName, "--namespace", namespace, "--ignore-not-found=true")
	
	// Wait a moment for resources to be fully deleted
	time.Sleep(2 * time.Second)

	defaultHelmArgs := []string{}
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultNeo4jNameArg...)
	defaultHelmArgs = append(defaultHelmArgs, "--wait", "--timeout", "300s")
	err := run(t, "helm", model.BaseHelmCommand("install", releaseName, chart, model.Neo4jEdition, defaultHelmArgs...)...)
	if err != nil {
		t.Log("Helm Install failed:", err)
		_ = run(t, "kubectl", "get", "events")
		return err
	}
	err = run(t, "kubectl", "--namespace", namespace, "rollout", "status", "--watch", "--timeout=120s", "statefulset/"+releaseName.String())
	if err != nil {
		t.Log("Helm Install failed:", err)
		return err
	}
	return err
}

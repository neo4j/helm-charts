package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetOurAnnotationsIgnoresGKEManagedTargetPool(t *testing.T) {
	t.Parallel()

	service := coreV1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"networking.gke.io/target-pool":        "gke-generated-target-pool",
				"networking.gke.io/load-balancer-type": "Internal",
				"cloud.google.com/neg-status":          "gke-generated-neg-status",
				"meta.helm.sh/release-name":            "test-release",
				"foo":                                  "bar",
			},
		},
	}

	assert.Equal(t, map[string]string{
		"networking.gke.io/load-balancer-type": "Internal",
		"foo":                                  "bar",
	}, getOurAnnotations(service))
}

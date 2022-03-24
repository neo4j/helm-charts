package integration_tests

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"testing"
)

//LabelNodes labels all the node with testLabel=<number>
func LabelNodes(t *testing.T) error {

	var errors *multierror.Error
	nodesList, err := getNodesList()
	if err != nil {
		return err
	}

	for index, node := range nodesList.Items {
		labelName := fmt.Sprintf("testLabel=%d", index+1)
		err = run(t, "kubectl", "label", "nodes", node.ObjectMeta.Name, labelName)
		if err != nil {
			errors = multierror.Append(errors, err)
			t.Logf("Node Label failed for %s: %v", node.ObjectMeta.Name, err)
		}
	}

	return errors.ErrorOrNil()
}

//RemoveLabelFromNodes removes label testLabel from all the nodes added via LabelNodes func
func RemoveLabelFromNodes(t *testing.T) error {

	var errors *multierror.Error
	nodesList, err := getNodesList()
	if err != nil {
		return err
	}

	for _, node := range nodesList.Items {
		err = run(t, "kubectl", "label", "nodes", node.ObjectMeta.Name, "testLabel-")
		if err != nil {
			errors = multierror.Append(errors, err)
			t.Logf("Node Label removal failed for %s: %v", node.ObjectMeta.Name, err)
		}
	}

	return errors.ErrorOrNil()
}

package codec

import (
	"bytes"
	"fmt"

	"k8s.io/kubernetes/pkg/api/unversioned"

	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/json"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
)

// UnstructuredToWorkflow transforms a runtime.Unstructured to a Workflow
func UnstructuredToWorkflow(unstruct *runtime.Unstructured) (*wapi.Workflow, error) {
	if unstruct.GetKind() != "Workflow" {
		return nil, fmt.Errorf("runtime.Unsstructured bad kind: %q", unstruct.GetKind())
	}
	buf := bytes.NewBuffer(nil)
	if err := runtime.UnstructuredJSONScheme.Encode(unstruct, buf); err != nil {
		return nil, fmt.Errorf("unable to encode unstructured")
	}
	w := &wapi.Workflow{}
	if err := json.Unmarshal(buf.Bytes(), w); err != nil {
		return nil, fmt.Errorf("unable to unmarshal: %v", err)
	}
	return w, nil
}

// WorkflowToUnstructured transforms a Workflow to runtime.Unstructured type
func WorkflowToUnstructured(workflow *wapi.Workflow, gvk *unversioned.GroupVersionKind) (*runtime.Unstructured, error) {
	rawJSON, err := json.Marshal(workflow)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal workflow")
	}
	obj, _, err := runtime.UnstructuredJSONScheme.Decode(rawJSON, gvk, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to decode rawJSON: %v", err)
	}
	unstruct, ok := obj.(*runtime.Unstructured)
	if !ok {
		return nil, fmt.Errorf("unable to convert workflow object")
	}
	return unstruct, nil
}

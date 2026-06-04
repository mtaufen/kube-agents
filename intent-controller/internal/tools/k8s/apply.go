package k8s

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type ApplyInput struct {
	Resource string                 `json:"resource"`
	Manifest map[string]interface{} `json:"manifest"`
}

type ApplyOutput struct {
	Message string `json:"message"`
}

// NewApplyTool creates an ADK tool for applying a Kubernetes resource.
func NewApplyTool(c client.Client) (tool.Tool, error) {
	handler := func(ctx tool.Context, input ApplyInput) (ApplyOutput, error) {
		return handleApply(ctx, c, input)
	}

	return functiontool.New(functiontool.Config{
		Name:        "apply_resource",
		Description: "Creates or updates a Kubernetes resource using Server-Side Apply. The input requires the plural resource name and the manifest object.",
	}, handler)
}

func handleApply(ctx context.Context, c client.Client, input ApplyInput) (ApplyOutput, error) {
	u := &unstructured.Unstructured{Object: input.Manifest}
	gvk := u.GroupVersionKind()

	name := u.GetName()
	namespace := u.GetNamespace()

	// Use Server-Side Apply
	opts := []client.PatchOption{
		client.ForceOwnership,
		client.FieldOwner("intent-agent"),
	}

	err := c.Patch(ctx, u, client.Apply, opts...)
	if err != nil {
		return ApplyOutput{}, err
	}

	return ApplyOutput{Message: fmt.Sprintf("Successfully applied %s %s/%s", gvk.Kind, namespace, name)}, nil
}

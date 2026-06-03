package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type ApplyInput struct {
	Resource string                 `json:"resource" jsonschema:"description=The plural resource name for RBAC checks, e.g. 'deployments'."`
	Manifest map[string]interface{} `json:"manifest" jsonschema:"description=The full manifest of the resource."`
}

type ApplyOutput struct {
	Message string `json:"message"`
}

// NewApplyTool creates an ADK tool for applying a Kubernetes resource, subject to the provided limits.
func NewApplyTool(c client.Client, limits []rbacv1.PolicyRule) (tool.Tool, error) {
	handler := func(ctx tool.Context, input ApplyInput) (ApplyOutput, error) {
		u := &unstructured.Unstructured{Object: input.Manifest}
		gvk := u.GroupVersionKind()
		
		group := gvk.Group
		name := u.GetName()
		namespace := u.GetNamespace()

		if !IsAllowed(limits, group, input.Resource, name, "patch") && !IsAllowed(limits, group, input.Resource, name, "update") && !IsAllowed(limits, group, input.Resource, name, "create") {
			return ApplyOutput{}, fmt.Errorf("permission denied: intent policy does not allow apply (needs patch/update/create) on %s/%s/%s in namespace %s", group, input.Resource, name, namespace)
		}

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

	return functiontool.New(functiontool.Config{
		Name:        "apply_resource",
		Description: "Creates or updates a Kubernetes resource using Server-Side Apply. The input requires the plural resource name and the manifest object.",
	}, handler)
}

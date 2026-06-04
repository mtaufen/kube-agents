package k8s

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type DeleteInput struct {
	Ref ResourceRef `json:"ref"`
}

type DeleteOutput struct {
	Message string `json:"message"`
}

// NewDeleteTool creates an ADK tool for deleting a Kubernetes resource, subject to the provided limits.
func NewDeleteTool(c client.Client, limits []rbacv1.PolicyRule) (tool.Tool, error) {
	handler := func(ctx tool.Context, input DeleteInput) (DeleteOutput, error) {
		return handleDelete(ctx, c, limits, input)
	}

	return functiontool.New(functiontool.Config{
		Name:        "delete_resource",
		Description: "Deletes a single Kubernetes resource. The input requires specifying the API group, version, kind, plural resource name, namespace, and name.",
	}, handler)
}

func handleDelete(ctx context.Context, c client.Client, limits []rbacv1.PolicyRule, input DeleteInput) (DeleteOutput, error) {
	if !IsAllowed(limits, input.Ref.Group, input.Ref.Resource, input.Ref.Name, "delete") {
		return DeleteOutput{}, fmt.Errorf("permission denied: intent policy does not allow 'delete' on %s/%s/%s in namespace %s", input.Ref.Group, input.Ref.Resource, input.Ref.Name, input.Ref.Namespace)
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   input.Ref.Group,
		Version: input.Ref.Version,
		Kind:    input.Ref.Kind,
	})
	u.SetNamespace(input.Ref.Namespace)
	u.SetName(input.Ref.Name)

	err := c.Delete(ctx, u)
	if err != nil {
		return DeleteOutput{}, err
	}

	return DeleteOutput{Message: fmt.Sprintf("Successfully deleted %s %s/%s", input.Ref.Kind, input.Ref.Namespace, input.Ref.Name)}, nil
}

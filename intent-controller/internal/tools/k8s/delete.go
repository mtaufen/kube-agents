package k8s

import (
	"context"
	"fmt"

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

// NewDeleteTool creates an ADK tool for deleting a Kubernetes resource.
func NewDeleteTool(c client.Client) (tool.Tool, error) {
	handler := func(ctx tool.Context, input DeleteInput) (DeleteOutput, error) {
		return handleDelete(ctx, c, input)
	}

	return functiontool.New(functiontool.Config{
		Name:        "delete_resource",
		Description: "Deletes a single Kubernetes resource. The input requires specifying the API group, version, kind, plural resource name, namespace, and name.",
	}, handler)
}

func handleDelete(ctx context.Context, c client.Client, input DeleteInput) (DeleteOutput, error) {
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

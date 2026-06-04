package k8s

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type ResourceRef struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Resource  string `json:"resource"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type GetInput struct {
	Ref ResourceRef `json:"ref"`
}

type GetOutput struct {
	Object map[string]interface{} `json:"object,omitempty"`
}

// NewGetTool creates an ADK tool for getting a Kubernetes resource.
func NewGetTool(c client.Client) (tool.Tool, error) {
	handler := func(ctx tool.Context, input GetInput) (GetOutput, error) {
		return handleGet(ctx, c, input)
	}

	return functiontool.New(functiontool.Config{
		Name:        "get_resource",
		Description: "Gets a single Kubernetes resource. The input requires specifying the API group, version, kind, plural resource name, namespace, and name.",
	}, handler)
}

func handleGet(ctx context.Context, c client.Client, input GetInput) (GetOutput, error) {

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   input.Ref.Group,
		Version: input.Ref.Version,
		Kind:    input.Ref.Kind,
	})

	err := c.Get(ctx, client.ObjectKey{Namespace: input.Ref.Namespace, Name: input.Ref.Name}, u)
	if err != nil {
		return GetOutput{}, err
	}
	return GetOutput{Object: u.Object}, nil
}

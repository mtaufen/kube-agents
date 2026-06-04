package k8s

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type ListInput struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Resource  string `json:"resource"`
	Namespace string `json:"namespace,omitempty"`
}

type ListOutput struct {
	Items []map[string]interface{} `json:"items"`
}

// NewListTool creates an ADK tool for listing Kubernetes resources.
func NewListTool(c client.Client) (tool.Tool, error) {
	handler := func(ctx tool.Context, input ListInput) (ListOutput, error) {
		return handleList(ctx, c, input)
	}

	return functiontool.New(functiontool.Config{
		Name:        "list_resources",
		Description: "Lists Kubernetes resources. The input requires specifying the API group, version, List kind, plural resource name, and optional namespace.",
	}, handler)
}

func handleList(ctx context.Context, c client.Client, input ListInput) (ListOutput, error) {

	uList := &unstructured.UnstructuredList{}
	uList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   input.Group,
		Version: input.Version,
		Kind:    input.Kind,
	})

	var listOpts []client.ListOption
	if input.Namespace != "" {
		listOpts = append(listOpts, client.InNamespace(input.Namespace))
	}

	err := c.List(ctx, uList, listOpts...)
	if err != nil {
		return ListOutput{}, err
	}

	items := make([]map[string]interface{}, 0, len(uList.Items))
	for _, item := range uList.Items {
		items = append(items, item.Object)
	}
	return ListOutput{Items: items}, nil
}

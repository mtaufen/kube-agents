package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type ListInput struct {
	Group     string `json:"group" jsonschema:"description=The API group of the resource, e.g. 'apps' or '' for core."`
	Version   string `json:"version" jsonschema:"description=The API version, e.g. 'v1'."`
	Kind      string `json:"kind" jsonschema:"description=The kind of the resource list, e.g. 'DeploymentList' (Must be the List kind)."`
	Resource  string `json:"resource" jsonschema:"description=The plural resource name for RBAC checks, e.g. 'deployments'."`
	Namespace string `json:"namespace,omitempty" jsonschema:"description=The namespace to list within. Leave empty for all namespaces if allowed."`
}

type ListOutput struct {
	Items []map[string]interface{} `json:"items"`
}

// NewListTool creates an ADK tool for listing Kubernetes resources, subject to the provided limits.
func NewListTool(c client.Client, limits []rbacv1.PolicyRule) (tool.Tool, error) {
	handler := func(ctx tool.Context, input ListInput) (ListOutput, error) {
		// Pass an empty string for resourceName since this is a list operation
		if !IsAllowed(limits, input.Group, input.Resource, "", "list") {
			return ListOutput{}, fmt.Errorf("permission denied: intent policy does not allow 'list' on %s/%s in namespace '%s'", input.Group, input.Resource, input.Namespace)
		}

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

	return functiontool.New(functiontool.Config{
		Name:        "list_resources",
		Description: "Lists Kubernetes resources. The input requires specifying the API group, version, List kind, plural resource name, and optional namespace.",
	}, handler)
}

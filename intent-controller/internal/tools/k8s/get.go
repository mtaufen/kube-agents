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

type ResourceRef struct {
	Group     string `json:"group" jsonschema:"description=The API group of the resource, e.g. 'apps' or '' for core."`
	Version   string `json:"version" jsonschema:"description=The API version, e.g. 'v1'."`
	Kind      string `json:"kind" jsonschema:"description=The kind of the resource, e.g. 'Deployment'."`
	Resource  string `json:"resource" jsonschema:"description=The plural resource name for RBAC checks, e.g. 'deployments'."`
	Namespace string `json:"namespace" jsonschema:"description=The namespace of the resource. Leave empty for cluster-scoped resources."`
	Name      string `json:"name" jsonschema:"description=The name of the resource."`
}

type GetInput struct {
	Ref ResourceRef `json:"ref"`
}

type GetOutput struct {
	Object map[string]interface{} `json:"object,omitempty"`
}

// NewGetTool creates an ADK tool for getting a Kubernetes resource, subject to the provided limits.
func NewGetTool(c client.Client, limits []rbacv1.PolicyRule) (tool.Tool, error) {
	handler := func(ctx tool.Context, input GetInput) (GetOutput, error) {
		if !IsAllowed(limits, input.Ref.Group, input.Ref.Resource, input.Ref.Name, "get") {
			return GetOutput{}, fmt.Errorf("permission denied: intent policy does not allow 'get' on %s/%s/%s in namespace %s", input.Ref.Group, input.Ref.Resource, input.Ref.Name, input.Ref.Namespace)
		}

		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   input.Ref.Group,
			Version: input.Ref.Version,
			Kind:    input.Ref.Kind,
		})

		// tool.Context implements context.Context in adk-go
		err := c.Get(ctx, client.ObjectKey{Namespace: input.Ref.Namespace, Name: input.Ref.Name}, u)
		if err != nil {
			return GetOutput{}, err
		}
		return GetOutput{Object: u.Object}, nil
	}

	return functiontool.New(functiontool.Config{
		Name:        "get_resource",
		Description: "Gets a single Kubernetes resource. The input requires specifying the API group, version, kind, plural resource name, namespace, and name.",
	}, handler)
}

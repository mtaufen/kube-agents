package k8s

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleList(t *testing.T) {
	scheme := runtime.NewScheme()

	pod1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod-1",
				"namespace": "default",
			},
		},
	}
	pod2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod-2",
				"namespace": "default",
			},
		},
	}

	// Fake client builder needs lists registered if we are fetching UnstructuredList
	// Or we can just use WithRuntimeObjects and rely on unstructured fallback
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(pod1, pod2).Build()

	limits := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"list"},
			APIGroups: []string{""},
			Resources: []string{"pods"},
		},
	}

	tests := []struct {
		name        string
		input       ListInput
		expectError bool
		expectCount int
	}{
		{
			name: "success list",
			input: ListInput{
				Group:     "",
				Version:   "v1",
				Kind:      "PodList",
				Resource:  "pods",
				Namespace: "default",
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "list error",
			input: ListInput{
				Group:     "invalid",
				Version:   "v1",
				Kind:      "PodList",
				Resource:  "pods", // matched by policy but will fail in client
				Namespace: "default",
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name: "denied list",
			input: ListInput{
				Group:     "apps",
				Version:   "v1",
				Kind:      "DeploymentList",
				Resource:  "deployments",
				Namespace: "default",
			},
			expectError: true,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := handleList(context.Background(), client, limits, tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("handleList() error = %v, expectError %v", err, tt.expectError)
			}
			if err == nil && len(out.Items) != tt.expectCount {
				t.Errorf("handleList() count = %v, expect %v", len(out.Items), tt.expectCount)
			}
		})
	}
}

func TestNewListTool(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	tool, err := NewListTool(client, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.Name() != "list_resources" {
		t.Errorf("unexpected name: %s", tool.Name())
	}
}

package k8s

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleDelete(t *testing.T) {
	scheme := runtime.NewScheme()

	existingObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod",
				"namespace": "default",
			},
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(existingObj).Build()

	limits := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"delete"},
			APIGroups: []string{""},
			Resources: []string{"pods"},
		},
	}

	tests := []struct {
		name        string
		input       DeleteInput
		expectError bool
	}{
		{
			name: "success allowed",
			input: DeleteInput{
				Ref: ResourceRef{
					Group:     "",
					Version:   "v1",
					Kind:      "Pod",
					Resource:  "pods",
					Namespace: "default",
					Name:      "test-pod",
				},
			},
			expectError: false,
		},
		{
			name: "delete error not found",
			input: DeleteInput{
				Ref: ResourceRef{
					Group:     "",
					Version:   "v1",
					Kind:      "Pod",
					Resource:  "pods",
					Namespace: "default",
					Name:      "missing-pod",
				},
			},
			expectError: true,
		},
		{
			name: "denied action",
			input: DeleteInput{
				Ref: ResourceRef{
					Group:     "apps",
					Version:   "v1",
					Kind:      "Deployment",
					Resource:  "deployments",
					Namespace: "default",
					Name:      "test-dep",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handleDelete(context.Background(), client, limits, tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("handleDelete() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestNewDeleteTool(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	tool, err := NewDeleteTool(client, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.Name() != "delete_resource" {
		t.Errorf("unexpected name: %s", tool.Name())
	}
}

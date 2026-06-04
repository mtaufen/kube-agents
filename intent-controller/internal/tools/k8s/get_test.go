package k8s

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleGet(t *testing.T) {
	scheme := runtime.NewScheme()

	existingObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":  "nginx",
						"image": "nginx",
					},
				},
			},
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(existingObj).Build()

	limits := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get"},
			APIGroups: []string{""},
			Resources: []string{"pods"},
		},
	}

	tests := []struct {
		name        string
		input       GetInput
		expectError bool
	}{
		{
			name: "success allowed",
			input: GetInput{
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
			name: "denied action",
			input: GetInput{
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
		{
			name: "not found allowed",
			input: GetInput{
				Ref: ResourceRef{
					Group:     "",
					Version:   "v1",
					Kind:      "Pod",
					Resource:  "pods",
					Namespace: "default",
					Name:      "missing-pod",
				},
			},
			expectError: true, // returns not found err
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handleGet(context.Background(), client, limits, tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("handleGet() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestNewGetTool(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	tool, err := NewGetTool(client, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.Name() != "get_resource" {
		t.Errorf("unexpected name: %s", tool.Name())
	}
}

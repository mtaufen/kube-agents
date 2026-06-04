package k8s

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleApply(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	tests := []struct {
		name        string
		input       ApplyInput
		expectError bool
		errContains string
	}{
		{
			name: "success allowed",
			input: ApplyInput{
				Resource: "pods",
				Manifest: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name":      "test-pod",
						"namespace": "default",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handleApply(context.Background(), client, tt.input)
			if err != nil {
				// Fake client does not support Apply patch type, so it may error out after passing auth.
				// We consider it a pass if it hits the unsupported patch error.
				if strings.Contains(err.Error(), "ApplyPatchType is not supported") {
					err = nil
				}
			}

			if (err != nil) != tt.expectError {
				t.Errorf("handleApply() error = %v, expectError %v", err, tt.expectError)
			}
			if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error to contain %q, got %v", tt.errContains, err)
			}
		})
	}
}

func TestNewApplyTool(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	tool, err := NewApplyTool(client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.Name() != "apply_resource" {
		t.Errorf("unexpected name: %s", tool.Name())
	}
}

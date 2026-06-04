package k8s

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleLoadSkill(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-skill",
			Namespace: "default",
		},
		Data: map[string]string{
			"SKILL.md": "skill content",
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	allowedSkills := []corev1.LocalObjectReference{
		{Name: "my-skill"},
	}

	tests := []struct {
		name        string
		input       LoadSkillInput
		expectError bool
		expectStr   string
	}{
		{
			name: "success allowed",
			input: LoadSkillInput{
				SkillName: "my-skill",
			},
			expectError: false,
			expectStr:   "skill content",
		},
		{
			name: "get error missing cm",
			input: LoadSkillInput{
				SkillName: "missing-skill",
			},
			expectError: true,
		},
		{
			name: "denied not allowed",
			input: LoadSkillInput{
				SkillName: "other-skill",
			},
			expectError: true,
		},
	}

	allowedSkills = append(allowedSkills, corev1.LocalObjectReference{Name: "missing-skill"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := handleLoadSkill(context.Background(), client, "default", allowedSkills, tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("handleLoadSkill() error = %v, expectError %v", err, tt.expectError)
			}
			if err == nil && !strings.Contains(out.Content, tt.expectStr) {
				t.Errorf("expected output to contain %q", tt.expectStr)
			}
		})
	}
}

func TestNewLoadSkillTool(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	tool, err := NewLoadSkillTool(client, "default", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.Name() != "load_skill" {
		t.Errorf("unexpected name: %s", tool.Name())
	}
}

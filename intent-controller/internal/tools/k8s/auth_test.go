package k8s

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestIsAllowed(t *testing.T) {
	tests := []struct {
		name         string
		limits       []rbacv1.PolicyRule
		apiGroup     string
		resource     string
		resourceName string
		verb         string
		expected     bool
	}{
		{
			name: "exact match",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get"},
					APIGroups: []string{"apps"},
					Resources: []string{"deployments"},
				},
			},
			apiGroup: "apps", resource: "deployments", resourceName: "my-dep", verb: "get",
			expected: true,
		},
		{
			name: "wildcard verb",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"*"},
					APIGroups: []string{"apps"},
					Resources: []string{"deployments"},
				},
			},
			apiGroup: "apps", resource: "deployments", resourceName: "my-dep", verb: "delete",
			expected: true,
		},
		{
			name: "wildcard group",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get"},
					APIGroups: []string{"*"},
					Resources: []string{"deployments"},
				},
			},
			apiGroup: "networking.k8s.io", resource: "deployments", resourceName: "my-dep", verb: "get",
			expected: true,
		},
		{
			name: "wildcard resource",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get"},
					APIGroups: []string{"apps"},
					Resources: []string{"*"},
				},
			},
			apiGroup: "apps", resource: "statefulsets", resourceName: "my-sts", verb: "get",
			expected: true,
		},
		{
			name: "exact resource name match",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:         []string{"get"},
					APIGroups:     []string{"apps"},
					Resources:     []string{"deployments"},
					ResourceNames: []string{"my-dep"},
				},
			},
			apiGroup: "apps", resource: "deployments", resourceName: "my-dep", verb: "get",
			expected: true,
		},
		{
			name: "wrong resource name",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:         []string{"get"},
					APIGroups:     []string{"apps"},
					Resources:     []string{"deployments"},
					ResourceNames: []string{"other-dep"},
				},
			},
			apiGroup: "apps", resource: "deployments", resourceName: "my-dep", verb: "get",
			expected: false,
		},
		{
			name: "verb not allowed",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get"},
					APIGroups: []string{"apps"},
					Resources: []string{"deployments"},
				},
			},
			apiGroup: "apps", resource: "deployments", resourceName: "my-dep", verb: "list",
			expected: false,
		},
		{
			name: "group not allowed",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get"},
					APIGroups: []string{""},
					Resources: []string{"pods"},
				},
			},
			apiGroup: "apps", resource: "pods", resourceName: "my-pod", verb: "get",
			expected: false,
		},
		{
			name: "resource not allowed",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get"},
					APIGroups: []string{"apps"},
					Resources: []string{"deployments"},
				},
			},
			apiGroup: "apps", resource: "statefulsets", resourceName: "my-sts", verb: "get",
			expected: false,
		},
		{
			name: "multiple rules - second matches",
			limits: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get"},
					APIGroups: []string{""},
					Resources: []string{"pods"},
				},
				{
					Verbs:     []string{"list"},
					APIGroups: []string{"apps"},
					Resources: []string{"deployments"},
				},
			},
			apiGroup: "apps", resource: "deployments", resourceName: "", verb: "list",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAllowed(tt.limits, tt.apiGroup, tt.resource, tt.resourceName, tt.verb); got != tt.expected {
				t.Errorf("IsAllowed() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

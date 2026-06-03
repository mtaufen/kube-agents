package k8s

import (
	rbacv1 "k8s.io/api/rbac/v1"
)

// IsAllowed checks if an action is allowed by a list of limits.
// For Phase 1, this is a very basic evaluation of rbacv1.PolicyRule.
//
// TODO(user): Replace this simple evaluation loop with official Kubernetes RBAC
// validation libraries (e.g., k8s.io/component-helpers/auth/rbac/validation.Covers)
// to ensure full compliance with K8s wildcard mechanics and edge cases.
func IsAllowed(limits []rbacv1.PolicyRule, apiGroup, resource, resourceName, verb string) bool {
	for _, rule := range limits {
		if !containsWildcardOrMatch(rule.Verbs, verb) {
			continue
		}
		if !containsWildcardOrMatch(rule.APIGroups, apiGroup) {
			continue
		}
		if !containsWildcardOrMatch(rule.Resources, resource) {
			continue
		}
		// If the rule specifies ResourceNames, the target resourceName must match one.
		if len(rule.ResourceNames) > 0 {
			if !containsWildcardOrMatch(rule.ResourceNames, resourceName) {
				continue
			}
		}
		return true // Match found
	}
	return false
}

func containsWildcardOrMatch(list []string, item string) bool {
	for _, s := range list {
		if s == "*" || s == item {
			return true
		}
	}
	return false
}

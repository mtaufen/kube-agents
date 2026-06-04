package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
	rbacv1 "k8s.io/api/rbac/v1"

	k8stools "kube-agents/intent-controller/internal/tools/k8s"
)

// compileAdaptivePolicy calls an LLM to generate an AdaptivePolicy (list of PolicyRules)
// bounded by the required and limits policies.
func compileAdaptivePolicy(ctx context.Context, apiKey, modelName, prompt string, required, limits []rbacv1.PolicyRule) ([]rbacv1.PolicyRule, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	reqJSON, _ := json.MarshalIndent(required, "", "  ")
	limJSON, _ := json.MarshalIndent(limits, "", "  ")

	systemInstruction := fmt.Sprintf(`You are an expert Kubernetes security architect.
Given the following user intent:
"%s"

Generate the MINIMUM necessary RBAC policy rules required to fulfill this intent.

Constraints:
1. You MUST include at least the permissions defined in the REQUIRED policy:
%s

2. You MUST NOT exceed the permissions defined in the LIMITS policy:
%s

Respond ONLY with a JSON array of Kubernetes rbac.v1.PolicyRule objects. Do not include markdown formatting or backticks.
Example:
[
  {
    "apiGroups": [""],
    "resources": ["pods"],
    "verbs": ["get", "list", "watch"]
  }
]`, prompt, reqJSON, limJSON)

	res, err := client.Models.GenerateContent(ctx, modelName, genai.Text(systemInstruction), &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate policy: %w", err)
	}

	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response candidates from LLM")
	}

	part := res.Candidates[0].Content.Parts[0]
	var text string
	if part.Text != "" {
		text = part.Text
	} else {
		return nil, fmt.Errorf("LLM response did not contain text")
	}

	// Sometimes the LLM might still wrap in markdown block despite instructions
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	// Parse the JSON string
	var generated []rbacv1.PolicyRule
	if err := json.Unmarshal([]byte(text), &generated); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w\nResponse was:\n%s", err, text)
	}

	return generated, nil
}

// verifyPolicy locally validates that the generated policy is a subset of limits and a superset of required.
func verifyPolicy(generated, required, limits []rbacv1.PolicyRule) error {
	if !covers(limits, generated) {
		return fmt.Errorf("generated policy exceeds limits")
	}
	if !covers(generated, required) {
		return fmt.Errorf("generated policy does not cover all required rules")
	}
	return nil
}

// covers checks if the covering policy fully allows all actions defined in the subset policy.
func covers(covering, subset []rbacv1.PolicyRule) bool {
	for _, rule := range subset {
		if !coversRule(covering, rule) {
			return false
		}
	}
	return true
}

// coversRule does a naive expansion of a single PolicyRule to verify if the covering policy allows it.
// This relies on the simplistic k8stools.IsAllowed logic.
func coversRule(covering []rbacv1.PolicyRule, rule rbacv1.PolicyRule) bool {
	for _, v := range rule.Verbs {
		for _, a := range rule.APIGroups {
			for _, r := range rule.Resources {
				if len(rule.ResourceNames) > 0 {
					for _, rn := range rule.ResourceNames {
						if !k8stools.IsAllowed(covering, a, r, rn, v) {
							return false
						}
					}
				} else {
					if !k8stools.IsAllowed(covering, a, r, "", v) {
						return false
					}
				}
			}
		}
	}
	return true
}

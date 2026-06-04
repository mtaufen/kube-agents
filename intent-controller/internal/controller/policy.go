package controller

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
	rbacv1 "k8s.io/api/rbac/v1"
)

// computePolicyHash generates a deterministic hash of the inputs that dictate the AdaptivePolicy.
func computePolicyHash(prompt string, required, limits []rbacv1.PolicyRule) string {
	hashData := struct {
		Prompt   string              `json:"prompt"`
		Required []rbacv1.PolicyRule `json:"required"`
		Limits   []rbacv1.PolicyRule `json:"limits"`
	}{
		Prompt:   prompt,
		Required: required,
		Limits:   limits,
	}
	b, _ := json.Marshal(hashData)
	return fmt.Sprintf("%x", sha256.Sum256(b))
}

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

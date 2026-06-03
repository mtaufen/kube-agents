package k8s

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type LoadSkillInput struct {
	SkillName string `json:"skill_name" jsonschema:"description=The name of the skill to load."`
}

type LoadSkillOutput struct {
	Content string `json:"content"`
}

// NewLoadSkillTool creates a tool that allows the agent to fetch the full text of an allowed skill ConfigMap.
func NewLoadSkillTool(c client.Client, namespace string, allowedSkills []corev1.LocalObjectReference) (tool.Tool, error) {
	handler := func(ctx tool.Context, input LoadSkillInput) (LoadSkillOutput, error) {
		allowed := false
		for _, s := range allowedSkills {
			if s.Name == input.SkillName {
				allowed = true
				break
			}
		}
		if !allowed {
			return LoadSkillOutput{}, fmt.Errorf("skill '%s' is not in the Intent's allowed skills list", input.SkillName)
		}

		var cm corev1.ConfigMap
		if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: input.SkillName}, &cm); err != nil {
			return LoadSkillOutput{}, err
		}

		var content string
		for key, val := range cm.Data {
			content += fmt.Sprintf("\n--- File: %s ---\n%s\n", key, val)
		}
		return LoadSkillOutput{Content: content}, nil
	}

	return functiontool.New(functiontool.Config{
		Name:        "load_skill",
		Description: "Loads the full content/instructions of a specific skill by its name.",
	}, handler)
}

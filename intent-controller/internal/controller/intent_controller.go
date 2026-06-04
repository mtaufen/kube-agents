/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	agentsv1alpha1 "kube-agents/intent-controller/api/v1alpha1"
	k8stools "kube-agents/intent-controller/internal/tools/k8s"
)

// IntentReconciler reconciles a Intent object
type IntentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=agents.gke.io,resources=intents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agents.gke.io,resources=intents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agents.gke.io,resources=intents/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Intent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *IntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	// Fetch the Intent instance
	var intent agentsv1alpha1.Intent
	if err := r.Get(ctx, req.NamespacedName, &intent); err != nil {
		logger.Error(err, "unable to fetch Intent")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling Intent", "name", intent.Name, "prompt", intent.Spec.Prompt)

	// TODO(Phase 2): Compile AdaptivePolicy using an LLM based on intent.Spec.Prompt.
	// For Phase 1, we will just use intent.Spec.Policy.Limits directly in our tools.

	// Setup model (e.g., Gemini) for the agent.
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	model, err := gemini.NewModel(ctx, "gemini-3.1-flash-lite", &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		logger.Error(err, "Failed to create gemini model")
		return ctrl.Result{}, err
	}

	// TODO(user): We currently provide specific tools (get, list, apply, delete) to improve
	// LLM function-calling reliability via strict JSON schemas. Consider adding or migrating to a generic
	// `call_kubernetes_api` tool if we need the LLM to access subresources (like /scale),
	// logs, or arbitrary endpoints, trading strict schemas for maximum flexibility.

	// Instantiate the k8s tools bounded by the static Intent Policy Limits
	getTool, err := k8stools.NewGetTool(r.Client, intent.Spec.Policy.Limits)
	if err != nil {
		logger.Error(err, "Failed to create get_resource tool")
		return ctrl.Result{}, err
	}
	listTool, err := k8stools.NewListTool(r.Client, intent.Spec.Policy.Limits)
	if err != nil {
		logger.Error(err, "Failed to create list_resources tool")
		return ctrl.Result{}, err
	}
	applyTool, err := k8stools.NewApplyTool(r.Client, intent.Spec.Policy.Limits)
	if err != nil {
		logger.Error(err, "Failed to create apply_resource tool")
		return ctrl.Result{}, err
	}
	deleteTool, err := k8stools.NewDeleteTool(r.Client, intent.Spec.Policy.Limits)
	if err != nil {
		logger.Error(err, "Failed to create delete_resource tool")
		return ctrl.Result{}, err
	}

	loadSkillTool, err := k8stools.NewLoadSkillTool(r.Client, intent.Namespace, intent.Spec.Skills)
	if err != nil {
		logger.Error(err, "Failed to create load_skill tool")
		return ctrl.Result{}, err
	}

	// Fetch ConfigMaps referenced in intent.Spec.Skills
	// and inject their frontmatter summaries into the agent's instructions.
	var skillsInstruction string
	for _, skillRef := range intent.Spec.Skills {
		var cm corev1.ConfigMap
		if err := r.Get(ctx, client.ObjectKey{Namespace: intent.Namespace, Name: skillRef.Name}, &cm); err != nil {
			logger.Error(err, "unable to fetch skill ConfigMap", "ConfigMap", skillRef.Name)
			// Requeue if a referenced skill ConfigMap is missing
			return ctrl.Result{}, err
		}

		// Attempt to extract frontmatter from SKILL.md or fallback to configmap name
		name := skillRef.Name
		desc := "Use load_skill to view."

		for _, val := range cm.Data {
			if n, d := parseFrontmatter(val); n != "" {
				name = n
				desc = d
				break
			}
		}
		skillsInstruction += fmt.Sprintf("- **%s**: %s (use `load_skill` to read full instructions)\n", name, desc)
	}

	baseInstruction := "Your goal is to fulfill the user's prompt by managing Kubernetes resources. You must use the provided tools to interact with the cluster."
	if skillsInstruction != "" {
		baseInstruction += "\n\nAvailable Skills:\n" + skillsInstruction
	}

	// Answer(AI): It is usually better to keep the `Instruction` as the "System Prompt" (setting the rules and identity),
	// and pass the `intent.Spec.Prompt` as the "User Prompt" when calling `agent.Run()`.
	// Initialize the adk-go agent
	adkAgent, err := llmagent.New(llmagent.Config{
		Name:        "intent_agent",
		Model:       model,
		Description: "An agent that actuates Kubernetes resources based on user Intent.",
		Instruction: baseInstruction,
		Tools: []tool.Tool{
			getTool,
			listTool,
			applyTool,
			deleteTool,
			loadSkillTool,
		},
	})
	if err != nil {
		logger.Error(err, "Failed to create agent")
		return ctrl.Result{}, err
	}

	// Run the agent
	rnr, err := runner.New(runner.Config{
		Agent:          adkAgent,
		SessionService: session.InMemoryService(),
	})
	if err != nil {
		logger.Error(err, "Failed to create runner")
		return ctrl.Result{}, err
	}

	meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{
		Type:    "Progressing",
		Status:  metav1.ConditionTrue,
		Reason:  "AgentRunning",
		Message: "The agent is actively evaluating and actuating the intent",
	})
	if err := r.Status().Update(ctx, &intent); err != nil {
		logger.Error(err, "Failed to update Intent status")
		return ctrl.Result{}, err
	}

	msg := &genai.Content{
		Parts: []*genai.Part{{Text: intent.Spec.Prompt}},
	}
	res := rnr.Run(ctx, intent.Namespace, intent.Name, msg, agent.RunConfig{})
	for event, err := range res {
		if err != nil {
			logger.Error(err, "Agent run encountered an error")
			meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{
				Type:    "Degraded",
				Status:  metav1.ConditionTrue,
				Reason:  "AgentError",
				Message: fmt.Sprintf("Agent run failed: %v", err),
			})
			_ = r.Status().Update(ctx, &intent)
			return ctrl.Result{}, err
		}
		// In a production scenario, we could log or store intermediate events
		_ = event
	}

	meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "AgentSucceeded",
		Message: "The agent has completed its execution successfully",
	})
	meta.RemoveStatusCondition(&intent.Status.Conditions, "Progressing")
	meta.RemoveStatusCondition(&intent.Status.Conditions, "Degraded")
	if err := r.Status().Update(ctx, &intent); err != nil {
		logger.Error(err, "Failed to update Intent status")
		return ctrl.Result{}, err
	}

	// We return a small RequeueAfter to allow the agent to continuously poll/manage
	// the resources, acting as a control loop. (In Phase 4, we'll replace this with
	// dynamic watches).
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// Answer(AI): `SetupWithManager` runs once at startup, so it's not the right place for per-Intent dynamic watches.
// We will keep `SetupWithManager` focused on watching the `Intent` resources themselves (and maybe the `Skills` ConfigMaps).
// For Phase 4 (dynamic watches), we will likely need a dynamic informer mechanism or a custom `source.Channel`
// that we configure dynamically from inside the `Reconcile` loop once the LLM decides what to watch.
//
// SetupWithManager sets up the controller with the Manager.
func (r *IntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentsv1alpha1.Intent{}).
		Named("intent").
		Complete(r)
}

func parseFrontmatter(content string) (string, string) {
	name, desc := "", ""
	parts := strings.SplitN(content, "---", 3)
	if len(parts) >= 3 {
		lines := strings.Split(parts[1], "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "name:") {
				name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			} else if strings.HasPrefix(line, "description:") {
				desc = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			}
		}
	}
	return name, desc
}

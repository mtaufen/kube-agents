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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"google.golang.org/adk/agent/llmagent"

	agentsv1alpha1 "kube-agents/intent-controller/api/v1alpha1"
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

	// TODO(Phase 1): Setup model (e.g., Gemini) for the agent.
	// model, err := gemini.NewModel(...)

	// Answer(AI): It is usually better to keep the `Instruction` as the "System Prompt" (setting the rules and identity),
	// and pass the `intent.Spec.Prompt` as the "User Prompt" when calling `agent.Run()`.
	// Initialize the adk-go agent
	agent, err := llmagent.New(llmagent.Config{
		Name: "intent_agent",
		// Model:       model,
		Description: "An agent that actuates Kubernetes resources based on user Intent.",
		Instruction: "Your goal is to fulfill the user's prompt by managing Kubernetes resources.",
		// Tools: []tool.Tool{ ... }, // TODO(Phase 1): Add client-go tools here
	})
	if err != nil {
		logger.Error(err, "Failed to create agent")
		return ctrl.Result{}, err
	}

	// Just to prevent unused variable error while scaffolding
	_ = agent

	// TODO(Phase 1): Fetch ConfigMaps referenced in intent.Spec.Skills
	// and inject them into the agent's instructions or tools.

	// TODO(Phase 1): Run the agent
	// res, err := agent.Run(ctx, intent.Spec.Prompt)

	return ctrl.Result{}, nil
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

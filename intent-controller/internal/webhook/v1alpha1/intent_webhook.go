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

package v1alpha1

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	agentsv1alpha1 "kube-agents/intent-controller/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var intentlog = logf.Log.WithName("intent-resource")

// SetupIntentWebhookWithManager registers the webhook for Intent in the manager.
func SetupIntentWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &agentsv1alpha1.Intent{}).
		WithValidator(&IntentCustomValidator{}).
		WithDefaulter(&IntentCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-agents-gke-io-v1alpha1-intent,mutating=true,failurePolicy=fail,sideEffects=None,groups=agents.gke.io,resources=intents,verbs=create;update,versions=v1alpha1,name=mintent-v1alpha1.kb.io,admissionReviewVersions=v1

// IntentCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Intent when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type IntentCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Intent.
func (d *IntentCustomDefaulter) Default(ctx context.Context, obj *agentsv1alpha1.Intent) error {
	intentlog.Info("Defaulting for Intent", "name", obj.GetName())

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		intentlog.Error(err, "Failed to get admission request from context")
		return nil // Return nil so we don't block operations without an admission context (e.g. envtest)
	}

	// Immutably set UserInfo to the actual user making the request.
	// This prevents privilege escalation by spoofing the user identity.
	obj.Spec.UserInfo = req.UserInfo

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-agents-gke-io-v1alpha1-intent,mutating=false,failurePolicy=fail,sideEffects=None,groups=agents.gke.io,resources=intents,verbs=create;update,versions=v1alpha1,name=vintent-v1alpha1.kb.io,admissionReviewVersions=v1

// IntentCustomValidator struct is responsible for validating the Intent resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type IntentCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Intent.
func (v *IntentCustomValidator) ValidateCreate(ctx context.Context, obj *agentsv1alpha1.Intent) (admission.Warnings, error) {
	intentlog.Info("Validation for Intent upon creation", "name", obj.GetName())

	return validateUserInfo(ctx, obj)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Intent.
func (v *IntentCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *agentsv1alpha1.Intent) (admission.Warnings, error) {
	intentlog.Info("Validation for Intent upon update", "name", newObj.GetName())

	return validateUserInfo(ctx, newObj)
}

func validateUserInfo(ctx context.Context, obj *agentsv1alpha1.Intent) (admission.Warnings, error) {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, nil // No admission context, skipping validation (e.g. testing)
	}

	// Defensive check: Ensure UserInfo was not spoofed or bypassed somehow
	if obj.Spec.UserInfo.Username != req.UserInfo.Username {
		return nil, fmt.Errorf("user info spoofing detected: requested username %q does not match actual username %q", obj.Spec.UserInfo.Username, req.UserInfo.Username)
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Intent.
func (v *IntentCustomValidator) ValidateDelete(_ context.Context, obj *agentsv1alpha1.Intent) (admission.Warnings, error) {
	intentlog.Info("Validation for Intent upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

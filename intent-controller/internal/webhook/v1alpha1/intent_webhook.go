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

	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"

	agentsv1alpha1 "kube-agents/intent-controller/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var intentlog = logf.Log.WithName("intent-resource")

// SetupIntentWebhookWithManager registers the webhook for Intent in the manager.
func SetupIntentWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &agentsv1alpha1.Intent{}).
		WithValidator(&IntentCustomValidator{Client: mgr.GetClient()}).
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
//
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
type IntentCustomValidator struct {
	Client client.Client
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Intent.
func (v *IntentCustomValidator) ValidateCreate(ctx context.Context, obj *agentsv1alpha1.Intent) (admission.Warnings, error) {
	intentlog.Info("Validation for Intent upon creation", "name", obj.GetName())

	if err := v.validatePermissions(ctx, obj); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Intent.
func (v *IntentCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *agentsv1alpha1.Intent) (admission.Warnings, error) {
	intentlog.Info("Validation for Intent upon update", "name", newObj.GetName())

	if err := v.validatePermissions(ctx, newObj); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Intent.
func (v *IntentCustomValidator) ValidateDelete(_ context.Context, obj *agentsv1alpha1.Intent) (admission.Warnings, error) {
	intentlog.Info("Validation for Intent upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

func (v *IntentCustomValidator) validatePermissions(ctx context.Context, obj *agentsv1alpha1.Intent) error {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil // No admission context, skipping validation (e.g. testing)
	}

	// Check both Limits and Required
	rules := append([]rbacv1.PolicyRule{}, obj.Spec.Policy.Limits...)
	rules = append(rules, obj.Spec.Policy.Required...)

	for _, rule := range rules {
		for _, verb := range rule.Verbs {
			for _, apiGroup := range rule.APIGroups {
				for _, resource := range rule.Resources {
					baseResource := resource
					subResource := ""
					if parts := strings.SplitN(resource, "/", 2); len(parts) == 2 {
						baseResource = parts[0]
						subResource = parts[1]
					}
					if len(rule.ResourceNames) > 0 {
						for _, resName := range rule.ResourceNames {
							if err := checkSAR(ctx, v.Client, req.UserInfo, obj.Namespace, verb, apiGroup, baseResource, subResource, resName); err != nil {
								return err
							}
						}
					} else {
						if err := checkSAR(ctx, v.Client, req.UserInfo, obj.Namespace, verb, apiGroup, baseResource, subResource, ""); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func checkSAR(ctx context.Context, k8sClient client.Client, userInfo authenticationv1.UserInfo, namespace, verb, group, resource, subresource, name string) error {
	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   userInfo.Username,
			Groups: userInfo.Groups,
			UID:    userInfo.UID,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Verb:        verb,
				Group:       group,
				Resource:    resource,
				Subresource: subresource,
				Name:        name,
				Namespace:   namespace, // Check permissions within the Intent's namespace
			},
		},
	}
	if len(userInfo.Extra) > 0 {
		sar.Spec.Extra = make(map[string]authorizationv1.ExtraValue)
		for k, v := range userInfo.Extra {
			sar.Spec.Extra[k] = authorizationv1.ExtraValue(v)
		}
	}

	if err := k8sClient.Create(ctx, sar); err != nil {
		return fmt.Errorf("failed to create SubjectAccessReview: %w", err)
	}
	if !sar.Status.Allowed {
		resStr := resource
		if subresource != "" {
			resStr = resource + "/" + subresource
		}
		return fmt.Errorf("user %q does not have permission to %s %s.%s %s in namespace %q", userInfo.Username, verb, resStr, group, name, namespace)
	}
	return nil
}

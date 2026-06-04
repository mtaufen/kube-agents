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
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IntentPolicy defines the policy bounds for the agent.
type IntentPolicy struct {
	// Limits define the maximum permissions the agent may be granted.
	// +optional
	Limits []rbacv1.PolicyRule `json:"limits,omitempty"`

	// Required defines the minimum permissions the agent must be granted.
	// +optional
	Required []rbacv1.PolicyRule `json:"required,omitempty"`
}

// IntentSpec defines the desired state of Intent
type IntentSpec struct {
	// Skills references ConfigMaps containing the agent's tools/instructions.
	// +optional
	Skills []corev1.LocalObjectReference `json:"skills,omitempty"`

	// Prompt contains the human-authored intent.
	Prompt string `json:"prompt"`

	// Model specifies the LLM to use (e.g., gemini-3.1-flash-lite).
	// Defaults to gemini-3.1-flash-lite if not specified.
	// +optional
	Model string `json:"model,omitempty"`

	// Policy defines the permission bounds for the agent.
	// +optional
	Policy IntentPolicy `json:"policy,omitempty"`

	// UserInfo contains information about the user who created the Intent.
	// +optional
	UserInfo authenticationv1.UserInfo `json:"userInfo,omitempty"`
}

// IntentStatus defines the observed state of Intent.
type IntentStatus struct {
	// AdaptivePolicy contains the dynamically compiled policy based on the prompt.
	// +optional
	AdaptivePolicy IntentPolicy `json:"adaptivePolicy,omitempty"`

	// PolicyHash stores the hash of the Spec inputs used to generate the AdaptivePolicy.
	// Used to bypass recompilation when the Spec has not changed.
	// +optional
	PolicyHash string `json:"policyHash,omitempty"`

	// conditions represent the current state of the Intent resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Intent is the Schema for the intents API
type Intent struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Intent
	// +required
	Spec IntentSpec `json:"spec"`

	// status defines the observed state of Intent
	// +optional
	Status IntentStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// IntentList contains a list of Intent
type IntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Intent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Intent{}, &IntentList{})
}

# Intent Controller Retrospective & Summary

This document summarizes the architecture, design decisions, and security model of the `intent-controller`, a Kubernetes operator that bridges natural language intents with autonomous cluster actuation using LLM agents.

## 1. Core Agent Actuation (Phase 1)

At the heart of the controller is the `Intent` Custom Resource Definition (CRD), which replaces traditional YAML specs with a natural language `Prompt`.

*   **ADK Integration:** The controller embeds the `google.golang.org/adk` agent framework directly inside the Kubernetes `Reconcile` loop.
*   **Tooling:** The agent is equipped with a suite of strict Kubernetes tools (`get_resource`, `list_resources`, `apply_resource`, `delete_resource`). It also has a `load_skill` tool, which allows it to fetch standard operating procedures stored in `ConfigMap` resources referenced in the `Intent.Spec.Skills` array.
*   **Execution:** During reconciliation, the agent reads the prompt, formulates a plan, and uses its tools to actuate the cluster state until the intent is fulfilled.

## 2. The Adaptive Policy Engine (Phase 2)

A massive challenge with AI agents in Kubernetes is **Least Privilege**. Granting a controller global `cluster-admin` privileges to execute open-ended prompts is a severe security risk.

To solve this, we implemented the **Adaptive Policy Engine**:
*   **Pre-Compilation:** Before the actuation agent runs, the controller passes the user's `Prompt` through an LLM to generate an `AdaptivePolicy`—a minimal, strictly necessary list of `rbacv1.PolicyRule`s required to fulfill the prompt.
*   **Hash Caching:** To prevent excessive LLM token burn and latency, the controller computes a SHA-256 `PolicyHash` based on the `Prompt`, `Required` rules, and `Limits`. Compilation is entirely bypassed on subsequent reconcile loops if the hash remains unchanged.

## 3. Security & Execution Isolation (Phase 3)

Even with an `AdaptivePolicy`, we needed to ensure the LLM couldn't hallucinate actions outside its bounds, and that malicious users couldn't use the controller for privilege escalation.

We implemented a two-part, bulletproof security model:

### A. Admission-Time Authorization
*   **The Webhook:** We implemented a Validating Admission Webhook that intercepts `Intent` creations and updates.
*   **SubjectAccessReview (SAR):** The webhook extracts the author's identity (`req.UserInfo`) and performs a strict `SubjectAccessReview` against the API server for every rule defined in `Intent.Spec.Policy.Limits`.
*   **Why Admission Time?** By enforcing that the author actually possesses the requested permissions at the moment of submission, we natively prevent privilege escalation. Furthermore, because authorization is resolved at admission rather than execution, an `Intent` won't suddenly start failing in production if the original author leaves the company or loses permissions.

### B. Ghost User Impersonation
*   **Dynamic RBAC:** Once authorized, the Reconciler dynamically creates a `Role` (containing the `AdaptivePolicy`) and a `RoleBinding` bound to a non-existent "Ghost User" URN (e.g., `kube-agents:intent:default:my-intent`).
*   **The Sandbox:** The agent's Kubernetes tools are instantiated with a `client-go` client configured to impersonate this Ghost User.
*   **The Result:** The Kubernetes API server acts as a flawless, native sandbox. If the LLM hallucinates an action outside its `AdaptivePolicy`, the API server rejects it with a standard `403 Forbidden` error, which the LLM can seamlessly read and correct. Additionally, K8s audit logs beautifully attribute the actions to the specific Intent URN rather than a generic controller service account.

## 4. Polling & Event Triggers (Phase 4)

While pure event-driven actuation (e.g., KRM Watches based on LLM output) is the theoretical ideal, it introduces extreme architectural complexity for MVP.

*   **Jittered Polling:** We opted for a pragmatic 1-minute polling interval utilizing a 10-second random jitter (`RequeueAfter: time.Minute + jitter`).
*   **Outcome:** This ensures the agent acts as a continuous control loop, constantly correcting cluster drift against the natural language intent, while the jitter prevents "thundering herd" scenarios that could throttle LLM API quotas.

## Conclusion

The `intent-controller` successfully demonstrates how to securely tether nondeterministic LLMs to rigid infrastructure. By pushing authorization to admission time and leveraging K8s-native Impersonation headers for execution, we achieve a system that is both incredibly flexible in its actuation and cryptographically secure in its boundaries.

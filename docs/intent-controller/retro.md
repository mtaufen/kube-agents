# Intent Controller Architecture Summary

The `intent-controller` is a Kubernetes operator that translates natural language intents into cluster state using LLM agents, built on the `google.golang.org/adk` framework.

## 1. Core API Definition (`Intent` CRD)
The `Intent` Custom Resource serves as the primary interface for users to define autonomous operations.

```yaml
apiVersion: agents.gke.io/v1alpha1
kind: Intent
metadata:
  name: example-intent
  namespace: intent-controller-system
spec:
  # The human-authored objective or goal.
  prompt: "Ensure a deployment named 'nginx' is running with 3 replicas."
  # (Optional) ConfigMaps containing standard operating procedures (SOPs).
  skills:
    - name: nginx-sops
  # Specifies the LLM to use (defaults to gemini-3.1-flash-lite)
  model: gemini-3.1-flash-lite
  # Defines the permission boundaries for the agent.
  policy:
    limits: # The absolute maximum permissions the agent can be granted
    - apiGroups: ["apps"]
      resources: ["deployments"]
      verbs: ["create", "update", "get", "list"]
    required: # The absolute minimum permissions the agent must be granted
    - apiGroups: [""]
      resources: ["pods"]
      verbs: ["get", "list"]
status:
  # The dynamically compiled minimal policy required to fulfill the prompt
  adaptivePolicy:
    limits: [...]
  # Hash of Spec inputs to bypass recompilation when unchanged
  policyHash: "sha256:..."
  conditions: [...]
```

## 2. Agent Execution Loop
- **Trigger**: The controller reconciles `Intent` CRDs. It uses a jittered 1-minute polling interval (`RequeueAfter: time.Minute + jitter`) to continually evaluate and correct cluster drift against the prompt.
- **Actuation**: The ADK agent reads `Intent.Spec.Prompt` and executes state changes using scoped tools (`get`, `list`, `apply`, `delete`).
- **Context**: Standard operating procedures are injected into the agent's instructions from ConfigMaps referenced in `Intent.Spec.Skills` (accessed via the `load_skill` tool).

## 3. Adaptive Policy Engine
- **Pre-Compilation**: To enforce Least Privilege, the controller pre-compiles the natural language prompt into an `AdaptivePolicy`—a minimal array of `rbacv1.PolicyRule`s required to fulfill the goal.
- **Caching**: The inputs (`Prompt`, `Required`, `Limits`) are combined into a SHA-256 `PolicyHash`. The LLM compilation step is bypassed if this hash is unchanged, significantly reducing token burn and latency.

## 4. Security & Isolation

### Admission Authorization & Structural Validation
- A Validating Admission Webhook intercepts `Intent` creations and updates.
- **Structural Checks**: The webhook performs strict structural validation of `rbacv1.PolicyRule` elements, rejecting empty verbs/resources and explicitly forbidding `nonResourceURLs` (since Intents are strictly namespace-scoped).
- **SubjectAccessReview (SAR)**: The webhook extracts the author's identity and performs a SAR check to verify the author possesses all permissions defined in the `Intent`'s `Limits` and `Required` arrays. This prevents privilege escalation. (Subresources like `pods/exec` are safely split and evaluated during this review).

### Ghost User Impersonation & Execution Sandboxing
- During execution, the controller dynamically provisions a `Role` (containing the `AdaptivePolicy`) and a `RoleBinding` bound to a deterministic "Ghost User" URN (`kube-agents:intent:<namespace>:<name>`). 
- The agent's tools use a `client-go` configuration that impersonates this Ghost User. 
- The Kubernetes API server acts as the native execution sandbox—blocking actions outside the `AdaptivePolicy` and accurately attributing operations in the cluster audit logs.

### Controller Pod Hardening & Network Defense
- **Unprivileged Execution**: The controller manager runs under a hardened Pod Security Context, strictly enforcing `runAsNonRoot: true`, explicitly binding to `runAsUser: 65532`, dropping all Linux capabilities, and using the `RuntimeDefault` seccomp profile.
- **Network Isolation**: NetworkPolicies enforce strict default-deny isolation on the controller's namespace, explicitly allow-listing ingress only for Prometheus metrics scraping (`metrics: enabled`) and Kubernetes API server webhook calls (allowing port 443 from outside the cluster network).

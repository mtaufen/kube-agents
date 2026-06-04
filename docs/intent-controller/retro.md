# Intent Controller Architecture Summary

The `intent-controller` is a Kubernetes operator that translates natural language intents into cluster state using LLM agents, built on the `google.golang.org/adk` framework.

## 1. Agent Execution Loop
- **Trigger**: The controller reconciles `Intent` CRDs. It uses a jittered 1-minute polling interval (`RequeueAfter: time.Minute + jitter`) to continually evaluate and correct cluster drift against the prompt.
- **Actuation**: The ADK agent reads `Intent.Spec.Prompt` and executes state changes using scoped tools (`get`, `list`, `apply`, `delete`).
- **Context**: Standard operating procedures are injected into the agent's instructions from ConfigMaps referenced in `Intent.Spec.Skills` (accessed via the `load_skill` tool).

## 2. Adaptive Policy Engine
- **Pre-Compilation**: To enforce Least Privilege, the controller pre-compiles the natural language prompt into an `AdaptivePolicy`—a minimal array of `rbacv1.PolicyRule`s required to fulfill the goal.
- **Caching**: The inputs (`Prompt`, `Required`, `Limits`) are combined into a SHA-256 `PolicyHash`. The LLM compilation step is bypassed if this hash is unchanged, significantly reducing token burn and latency.

## 3. Security & Isolation
- **Admission Authorization**: A Validating Admission Webhook intercepts `Intent` creations and updates. It extracts the author's identity and performs a `SubjectAccessReview` (SAR) to verify the author possesses all permissions defined in `Intent.Spec.Policy.Limits`. This resolves authorization at authoring time and prevents privilege escalation.
- **Ghost User Impersonation**: During execution, the controller dynamically provisions a `Role` (containing the `AdaptivePolicy`) and a `RoleBinding` bound to a deterministic "Ghost User" URN (`kube-agents:intent:<namespace>:<name>`). 
- **API Server Sandboxing**: The agent's tools use a `client-go` configuration that impersonates the Ghost User. The Kubernetes API server acts as a native execution sandbox—blocking actions outside the `AdaptivePolicy` and accurately attributing operations in the cluster audit logs.

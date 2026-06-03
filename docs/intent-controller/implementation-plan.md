# Intent Controller Implementation Plan

This plan outlines a phased approach to building the Intent Controller, prioritizing simplicity and robustness to get a working baseline before adding complex dynamic behaviors.

## Phase 1: Core API Simplification & Basic Actuation
*The goal here is to get a hard-coded agent loop reconciling a prompt, using statically defined permissions.*

1. **API Simplification**: 
   * Drop the `Agent` and `ReviewAgent` reference fields from `IntentSpec`.
   * Add a `Skills` field referencing a list of ConfigMaps containing the agent's tools/instructions. The "review gate" simply becomes a specific skill entrypoint within these ConfigMaps.
2. **Agent Harness Integration**: Integrate `adk-go` as the core execution engine within the `Reconcile` loop.
3. **LLM Tooling (client-go)**: Provide the LLM with `client-go` based tools to interact with the cluster. Initially, these tools will be strictly limited by the `Limits` policy statically defined in the Intent.
4. **Basic Reconcile Trigger**: Start with a simple, periodic reconcile (polling) or a broad watch. Avoid dynamic, per-Intent LLM-driven watches until the core actuation is rock solid.

## Phase 2: Adaptive Policy Engine
*The goal is to dynamically generate the minimum necessary permissions for the prompt and enforce them locally.*

1. **Policy Compiler Step**: Add an LLM call *before* the main actuation loop that reads the `Prompt` and generates an `AdaptivePolicy` (a list of RBAC rules).
2. **Local Policy Verification**: Use Kubernetes RBAC libraries (e.g., `k8s.io/apiserver/pkg/authorization/authorizer` or simple set math) to mathematically guarantee the generated `AdaptivePolicy`:
   * Does not exceed the rules defined in `Intent.Spec.Policy.Limits`.
   * Fully covers the rules defined in `Intent.Spec.Policy.Required`.
3. **Tool Sandbox**: Update the `client-go` tools provided to the LLM so they strictly enforce the `AdaptivePolicy`. Any API request outside this policy is rejected locally.

## Phase 3: Identity & Privilege Escalation Prevention
*The goal is to ensure a human cannot use an Intent to bypass their own RBAC restrictions.*

1. **Identity Injection**: Implement a Mutating Admission Webhook. When an `Intent` is created or updated, the webhook intercepts the request and immutably sets `Spec.UserInfo` to the actual Kubernetes user making the request.
2. **User Permission Validation**: Extend the local policy check (from Phase 2) to issue a `SubjectAccessReview`. Ensure the user in `UserInfo` actually has permission to perform every action in the `AdaptivePolicy`. If not, attenuate the policy or fail.
3. **Execution Isolation**: Instead of the controller executing actions as itself, dynamically create a dedicated Kubernetes ServiceAccount (KSA) for each Intent, bind the `AdaptivePolicy` to it, and have the agent impersonate that KSA.

## Phase 4: Advanced Intelligence & Performance
*The goal is to add the complex behaviors like review gates, dynamic watches, and memory.*

1. **Review Gate Step**: Before the agent applies state changes, route the proposed changes through a "Review Skill" (defined in the ConfigMaps) to analyze for security/compliance and either approve, modify, or block.
2. **Dynamic Watches**: Ask the LLM which resources it needs to monitor to fulfill the Intent. Dynamically spin up or modify informers/watches for those specific resources to trigger the work queue, replacing the basic polling from Phase 1.
3. **Persistent Memory**: Implement a mechanism for the agent to store its thought process, discovered context, and historical actions. This can be stored in a dedicated `ConfigMap` or `Secret` owned by and tied to the lifecycle of the `Intent`.
4. **Extended Policy Options**: Expand the policy engine beyond RBAC to include allowed external endpoints (e.g., fetching logs from an external observability stack, or metrics from Prometheus).

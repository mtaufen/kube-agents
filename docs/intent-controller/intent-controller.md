# Intent Controller

There are a few ways policy for agents can "adapt" to the situation at hand:

* **Generate a least-privilege policy based on a trusted prompt**, BEFORE an agent takes potentially dangerous actions or encounters untrusted content that may contain prompt injections.  
  * Protects agents and infrastructure by limiting future actions based on initial understanding of job role.  
  * Strongly resistant to prompt injection, because it's generated before injection can occur.  
  * In theory, if we limit policy extensions to exclusively *more restrictive* directions, we can even generate additional attenuations from untrusted context. Whether this is also useful is TBD; we can't safely generate *relaxations*.  
* Require each action to pass through a review gate that **automatically applies custom security and compliance adjustments** **to the agent's resource updates**, specific to the situation.  
  * Protects infrastructure by applying security and compliance rules.  
  * Opportunity to try to make it have "reasonable judgement" by understanding deployment context (like a human security reviewer), with some tradeoff on risk.   
  * Some chance it may be bypassed by indirect prompt injection (in the resources under review), but still valuable as   
  * Need to be careful that indirect injections don't escalate privilege (e.g. by creating broad RBAC rules when security policy is applied).

In addition: 

* **Upper bound:** It's a good idea to have some human-authored policy at the top level, providing an upper bound regardless of generated policy.  
* **Lower bound:** It *may* be a good idea to have some human-authored policy to establish a lower-bound on permissions, rather than always trusting the policy generator to allow permissions that are ALWAYS needed. Must be careful not to abuse this.  
* **Prevent escalation:** It's important to prevent privilege escalation of agents or humans on policy paths. For example, a human shouldn't be able to escalate access just by setting a more permissive policy on an agent and then sending it a prompt.

# Sketch: How this could work {#sketch:-how-this-could-work}

I'm going to take some liberties here and describe a new API, to illustrate how this could work. The new API is called *Intent*.

## Human-authored: Each agent gets a separate KSA (firm upper bound) {#human-authored:-each-agent-gets-a-separate-ksa-(firm-upper-bound)}

Each agent definition could be given its own KSA, and RBAC permissions could be bound to the KSA to create an upper bound on the agent. This is similar to GCP agent identity, where the identity is used across a "class" of agent.

```
apiVersion: v1
kind: ServiceAccount
metadata:
  name: devteam
  namespace: my-namespace
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: read-pods
  namespace: my-namespace
subjects:
- kind: ServiceAccount
  name: devteam
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role # or ClusterRole if there is one global devteam agent?
  name: devteam
  apiGroup: rbac.authorization.k8s.io
```

## Human-authored: Intent, per-intent policy  {#human-authored:-intent,-per-intent-policy}

All Kubernetes resources represent a statement of intent that is *continuously reconciled* by a controller. Shouldn't agentic operation follow the same pattern? Consider a new custom resource, simply called *Intent*, that can represent *any* intent:

* *Intent*: Namespaced. Binds together an agent definition reference, user prompt, policy, and info about the user who created the intent. The Intent is reconciled by a controller that continuously watches relevant resources (determined by LLM within bounds of policy) and responds to changes by asking an LLM to generate an appropriate set of requests to make against kube-apiserver. These requests are bounded by both the RBAC applied to the agent's KSA and the   
* *ClusterIntent*: Like *Intent*, but cluster-scope/used for non-namespaced resources like Nodes.

### Per-Intent policy (upper and lower bounds) {#per-intent-policy-(upper-and-lower-bounds)}

When a human writes a particular prompt, they probably have some idea of the appropriate upper bound for that *specific task*, which is likely more limited than the whole devteam agent. They may also have some idea of the absolutely required permissions that are needed for that task. They can specify this as part of the *Intent*.

### Preventing human escalation {#preventing-human-escalation}

The devteam agent may have more permission than the user (or possibly agent) who created the Intent. Therefore, the user must not be able to get the agent to do something that they don't already have permission to do. Info about the user who created the intent should be injected by the API (or admission hook) at creation time, and be immutable afterwards, similar to [CertificateSigningRequests](https://kubernetes.io/docs/reference/kubernetes-api/certificates/certificate-signing-request-v1/#CertificateSigningRequestSpec) and [AdmissionReviews](https://kubernetes.io/docs/reference/kubernetes-api/definitions/subject-access-review-v1-authorization/#SubjectAccessReviewSpec), so that agent actions are also restricted to the upper bound of the requestor's permissions. Beyond the RBAC permission to create Intent objects, this establishes that the prompt in the Intent can be trusted to the extent that it is unable to escalate beyond actions the user could already take. 

### Example Resource {#example-resource}

```yaml
apiVersion: agents.gke.io/v1alpha1
kind: Intent
metadata:
  name: my-deployment-optimizer
  namespace: my-namespace
spec:
  agent: devteam
  reviewAgent: security-and-compliance
  prompt: "Deploy Grafana to monitor my-namespace and keep Grafana running."
  policy: # could be formatted as RBAC rules, VAP, etc, checked locally in controller.
    limits: # the agent MAY be granted these, but may also recuse itself from them
    - apiGroups: [""] # "" indicates the core API group
      resources: ["pods/log"]
      verbs: ["get"]
    - apiGroups: ["apps"]
      resources: ["deployments"]
      verbs: ["get", "list", "watch", "create", "update", "patch"]
    required: # the agent MUST be granted these on the same namespace as the Intent
    - apiGroups: [""] # "" indicates the core API group
      resources: ["pods"]
      verbs: ["get", "watch", "list"]
  userInfo:
    username: mtaufen@google.com
    groups: []
    extra: []
    uid: 014fbff9a07c
status:
  adaptivePolicy: # dynamically compiled policy included in status for debugging
    limits: 
    - apiGroups: [""] # "" indicates the core API group
      resources: ["pods/log"]
      verbs: ["get"]
    # policy complier applied additional resource name restriction based on context
    - apiGroups: ["apps"]
      resources: ["deployments"]
      verbs: ["get", "list", "watch", "create", "update", "patch"]
      resourceNames: ["grafana"]
    required:
    - apiGroups: [""] # "" indicates the core API group
      resources: ["pods"]
      verbs: ["get", "watch", "list"]
```

## Auto-generated: Adaptive Policy {#auto-generated:-adaptive-policy}

Before invoking the LLM to enact the Intent, the Intent controller queries an *AI* *policy compiler* that makes an LLM call to determine an appropriate *adaptive policy* based on the prompt. The Intent controller then deterministically adjusts the returned policy by:

* Stripping any rules that exceed *policy.limits*.  
* Ensuring *policy.required* is still represented in the rules.  
* Sending a SubjectAccessReview to ensure the user in *userInfo* can actually perform all the actions allowed by the policy. If not, the policy is further attenuated until it matches what the user can actually do.  
  * Note: It is not wise to repeat this on every reconcile. The user's permissions may change in the future, but it's best to just take into account their permissions at the time the Intent was created. Humans changing teams or leaving companies shouldn't cause Intent outages. 

## Auto-generated: Review gate for deployed resources {#auto-generated:-review-gate-for-deployed-resources}

Finally, the Intent also includes a *reviewAgent* that can recommend additional security improvements (within the bounds of what's allowed by the Intent policy to avoid privilege escalation) and block deployment if it doesn't pass review criteria. For example, if the Intent is only allowed to create Deployments, it can still lock down the SecurityContext of those Deployments based on *reviewAgent* recommendations.

**Caveat:** What if the human doesn't have permission to deploy e.g. RBAC or VAP or NetworkPolicy, but these policies if recommended by the reviewAgent are still much better for security if applied than not applied (or dangerous if not applied)? We prevent privilege escalation via prompt, but at the same time prevent automatic security improvement? Can we resolve this tradeoff? 

* Do we return an error in this case? Seems reasonable for reviewAgent to block, but also annoying.  
* reviewAgent *is* configurable, and admins can use their judgement in setting review criteria.  
* May come down to how much the user is trusted to create the generic Intent.  
* Maybe we want something more advanced, where admins can designate "safe policy extensions" where the policy is exclusively more restrictive than the status-quo in the bundle of proposed updates for a given Intent reconciliation. Maybe we can automatically determine this.

# Intent Reconciliation {#intent-reconciliation}

Sketch of the flow:

1. User creates Intent  
2. Controller sees new (or updated) Intent, adaptive policy is LLM-compiled based on Intent, limited by policy.limits and user's current permissions  
3. Controller queries LLM to determine which underlying resources to watch for this Intent  
4. Controller queries LLM to determine actions.  
5. Actions are filtered or rejected in controller based on adaptive policy.  
6. Requests are made based on actions.  
   1. Limited by agent KSA's RBAC  
   2. Additionally limited by any VAP or other admission policies on the cluster  
7. Repeat  
   


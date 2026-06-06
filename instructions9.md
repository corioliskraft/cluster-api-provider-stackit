# Instructions: Add Admission Webhooks

## Goal

Implement admission webhooks for the STACKIT infrastructure provider API.

The CRDs already contain structural schema and some CEL validation, but the
provider currently does not ship defaulting or validation webhooks. Add
webhooks so invalid or unsafe input fails early with clear Kubernetes field
errors, and so implicit controller defaults become explicit API defaults where
that is safe.

Prioritise correctness over speed. Read the code first, then implement the
smallest defensible webhook surface with strong tests.

## Required References

Use this repository as the source of truth for current API and controller
behaviour:

- `api/v1alpha1/stackitcluster_types.go`
- `api/v1alpha1/stackitmachine_types.go`
- `api/v1alpha1/stackitclustertemplate_types.go`
- `api/v1alpha1/stackitmachinetemplate_types.go`
- `internal/controller/stackitcluster_controller.go`
- `internal/controller/stackitmachine_controller.go`
- `cmd/main.go`
- `config/default/kustomization.yaml`
- `config/crd/kustomization.yaml`
- `config/rbac/capi_manager_aggregation_role.yaml`

Use the local Cluster API checkout as the main API reference:

```text
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/cluster-api
```

Recommended Cluster API webhook references:

- `bootstrap/kubeadm/internal/webhooks/kubeadmconfigtemplate.go`
- `controlplane/kubeadm/internal/webhooks/kubeadmcontrolplanetemplate.go`
- `controlplane/kubeadm/internal/webhooks/kubeadmcontrolplane.go`
- `internal/webhooks/cluster.go`
- `internal/webhooks/machine.go`
- `internal/webhooks/machineset.go`
- `internal/webhooks/clusterresourceset.go`

Important: Cluster API has helpers under `internal/`, for example
`internal/util/compare`. This provider cannot import those packages because Go
`internal` visibility forbids it. Reimplement small local helpers or use public
packages such as `k8s.io/apimachinery/pkg/api/equality` and
`github.com/google/go-cmp/cmp`.

## JJ Workflow

This repository uses Jujutsu. Do not use raw git for VCS operations.

Start with:

```sh
jj status
```

If the working copy is not already an empty change for this task, create one:

```sh
jj new -m "Add admission webhooks"
```

Keep this work in a dedicated JJ change. Do not mix it into prior docs,
ClusterClass, bastion, or addon changes. Use `jj diff --git` and `jj status`
frequently.

## What RBAC Aggregation Means

RBAC aggregation is Kubernetes' mechanism for composing ClusterRoles by label.
Cluster API's manager role selects ClusterRoles labeled:

```yaml
cluster.x-k8s.io/aggregate-to-manager: "true"
```

The provider already ships `config/rbac/capi_manager_aggregation_role.yaml`.
That role is not for the STACKIT provider manager. It is aggregated into the
Cluster API core manager role so CAPI core controllers can create, update,
patch, watch, and delete STACKIT infrastructure objects. This is required for
ClusterClass-generated infrastructure resources, owner references, and label
management.

Do not remove or weaken this role while adding webhooks. Webhook RBAC and CAPI
manager aggregation solve different problems:

- Admission webhooks validate/default incoming API requests.
- RBAC aggregation grants CAPI core controllers permission to manage provider
  resources.

## Implementation Plan

### 1. Audit Current API Behaviour

Before writing code, document what the current controllers actually support.
In particular inspect:

- `StackitCluster` reconciliation of:
  - `spec.projectID`
  - `spec.region`
  - `spec.credentialsSecretRef`
  - `spec.network.id`
  - `spec.apiServerLoadBalancer.enabled`
  - `spec.controlPlaneEndpoint`
  - `spec.bastion`
  - `spec.bastion.cloudInitRef`
- `StackitMachine` reconciliation of:
  - `spec.providerID`
  - `spec.imageID`
  - `spec.machineType`
  - `spec.availabilityZone`
  - `spec.sshKeyName`
  - `spec.rootVolume`
  - `spec.network.id`
  - `spec.securityGroups`
  - `spec.additionalLabels`
- Template handling for:
  - `StackitClusterTemplate.spec.template.metadata`
  - `StackitMachineTemplate.spec.template.metadata`
  - `spec.template.spec` immutability

Do not guess update safety. If a field is consumed only during create and is
not reconciled in-place later, treat updates to it as unsafe and reject them.
If a field is intentionally reconciled in-place, allow it and test it.

### 2. Scaffold Webhooks

Follow the repository's `AGENTS.md`: use Kubebuilder to scaffold webhooks
instead of creating all files by hand.

Scaffold validation/defaulting webhooks for:

```sh
kubebuilder create webhook --group infrastructure --version v1alpha1 --kind StackitCluster --defaulting --programmatic-validation
kubebuilder create webhook --group infrastructure --version v1alpha1 --kind StackitMachine --defaulting --programmatic-validation
kubebuilder create webhook --group infrastructure --version v1alpha1 --kind StackitClusterTemplate --defaulting --programmatic-validation
kubebuilder create webhook --group infrastructure --version v1alpha1 --kind StackitMachineTemplate --defaulting --programmatic-validation
```

After scaffolding, inspect all generated files. Preserve scaffold markers in
`cmd/main.go`, `PROJECT`, and kustomize files. Do not edit generated CRDs,
generated webhook manifests, or `zz_generated.deepcopy.go` manually.

Expected areas to wire:

- `internal/webhook/v1alpha1/*`
- `cmd/main.go` webhook setup calls
- `config/webhook/*`
- `config/certmanager/*` if scaffolded
- `config/default/kustomization.yaml` webhook/cert-manager patches
- `config/crd/kustomization.yaml` only if conversion webhooks are added; do not
  add conversion webhooks for this task unless a real version-conversion need is
  discovered

### 3. Implement Shared Validation Helpers

Create small, tested helpers in the webhook package instead of duplicating
logic across four webhook files.

Use Kubernetes-style field errors:

- Build `field.ErrorList`.
- Use precise paths, for example:
  - `field.NewPath("spec", "network", "id")`
  - `field.NewPath("spec", "template", "metadata")`
  - `field.NewPath("spec", "template", "spec")`
- Return `apierrors.NewInvalid(GroupVersion.WithKind(...).GroupKind(), name, allErrs)`.
- Prefer `field.Required`, `field.Invalid`, and `field.Forbidden` over plain
  `fmt.Errorf`.

Validation helpers to consider:

- `validateStackitClusterSpec(spec, path)`
- `validateStackitMachineSpec(spec, path, allowProviderID bool)`
- `validateRootVolume(spec, path)`
- `validateBastionSpec(spec, path)`
- `validateBastionCloudInitRef(ref, path)`
- `validateAdditionalLabels(labels, path)`
- `validateTemplateObjectMeta(metadata, path)`
- `validateImmutable(oldValue, newValue, path, message)`

Use existing CRD markers as a baseline, but webhooks should add clearer
messages and update-safety checks, not replace CRD validation.

### 4. Defaulting Rules

Only default values that already match current controller behaviour or are
clearly safe API defaults.

Recommended defaults to implement first:

- `StackitMachine.spec.rootVolume.deleteOnTermination`: default to `true` when
  `rootVolume` is used and the field is nil.
- `StackitCluster.spec.bastion.rootVolume.deleteOnTermination`: default to
  `true` when bastion is enabled or when a bastion root volume is configured
  and the field is nil.
- Template resources should apply the same defaults to
  `spec.template.spec`.

Be careful with `spec.apiServerLoadBalancer.enabled`. The zero value currently
means false, and the CRD requires `controlPlaneEndpoint.host` when the load
balancer is disabled. Do not silently change omitted load-balancer settings to
enabled unless you intentionally decide to change API semantics and update
tests/docs/templates accordingly.

Defaulting webhooks must be idempotent.

### 5. Validation Rules

Implement create validation for all four resource types.

#### StackitCluster

Validate at least:

- `spec.projectID`, `spec.region`, `spec.network.id`, and
  `spec.credentialsSecretRef.name` are set and valid.
- `spec.credentialsSecretRef.namespace` is either empty or valid. Before
  forbidding cross-namespace Secret refs, confirm controller behaviour:
  `secretData` currently defaults an empty namespace to the `StackitCluster`
  namespace but allows an explicit namespace.
- If `spec.apiServerLoadBalancer.enabled` is false, require
  `spec.controlPlaneEndpoint.host`.
- If `spec.apiServerLoadBalancer.enabled` is true, reject a user-supplied
  `spec.controlPlaneEndpoint` unless the controller explicitly supports
  reconciling both. The load balancer path owns the endpoint.
- If `spec.bastion.enabled` is true, require:
  - `imageID`
  - `machineType`
  - `sshKeyName`
  - at least one `allowedCIDRs` entry
- Validate `spec.bastion.cloudInitRef` when set:
  - `kind` must be `ConfigMap` or `Secret`
  - `name` and `key` must be non-empty
  - do not require the referenced object to exist on create; that breaks
    GitOps/apply ordering and the controller already reports this as a
    condition
- Validate root volume values and additional labels.

Validate update safety:

- Treat `spec.projectID`, `spec.region`, and `spec.network.id` as immutable.
- Treat `spec.credentialsSecretRef` as immutable unless you confirm credential
  ref switching is safe for all dependent resources.
- Treat `spec.apiServerLoadBalancer.enabled` as immutable after the load
  balancer has been provisioned, unless you first implement and test full
  create/delete/update semantics for toggling it.
- Allow supported bastion lifecycle changes:
  - enabling bastion from disabled when all required fields are present
  - disabling bastion, because the controller deletes provider-managed bastion
    resources
  - changing `cloudInitRef`, because the controller tracks
    `status.bastion.cloudInitHash` and recreates the bastion when the referenced
    content changes
- For other bastion fields, audit controller/cloud behaviour first. If a field
  is ignored after the bastion exists, reject updates after
  `status.bastion.serverID` is set with a clear message.

#### StackitMachine

Validate at least:

- `spec.imageID`, `spec.machineType`, and `spec.network.id` are set and valid.
- `spec.providerID`, if set, uses the provider format accepted by
  `pkg/cloud.ParseProviderID`.
- `spec.securityGroups` contains valid IDs and no duplicates.
- root volume values are coherent:
  - `sizeGiB` cannot be negative
  - `performanceClass`, if set, must be non-empty
  - if `deleteOnTermination` is set while no boot volume is requested, decide
    whether this is allowed or should be rejected as ineffective
- `additionalLabels` keys/values are Kubernetes-label-compatible or explicitly
  document why STACKIT labels allow a different format.

Validate update safety:

- Forbid user changes to VM creation fields after create:
  - `imageID`
  - `machineType`
  - `availabilityZone`
  - `sshKeyName`
  - `rootVolume`
  - `network.id`
  - `securityGroups` unless the controller/cloud client truly reconciles them
    in-place
- `spec.providerID` is controller-owned. Allow the nil-to-valid-providerID
  transition needed by the controller, allow no-op updates, and reject changing
  an existing providerID to a different value.
- Decide whether `additionalLabels` are immutable after provisioning. If the
  cloud client does not update labels on an existing server, reject changes
  after `status.instanceID` is set.

#### StackitClusterTemplate

Validate:

- `spec.template.metadata` with Cluster API's `clusterv1.ObjectMeta.Validate`
  pattern. See CAPI's `KubeadmConfigTemplate` and
  `KubeadmControlPlaneTemplate` webhooks.
- `spec.template.spec` with the same create validation used for
  `StackitClusterSpec`, but with field paths rooted at
  `spec.template.spec`.

Validate updates:

- `spec.template.spec` must be immutable. Follow the CAPI
  `KubeadmControlPlaneTemplate` model: if the template spec changes, reject the
  update and tell users to create a new template instead.
- Allow `spec.template.metadata.labels` and annotations to change if they pass
  metadata validation. CAPI templates commonly validate metadata but do not make
  it immutable.

#### StackitMachineTemplate

Validate:

- `spec.template.metadata` using `clusterv1.ObjectMeta.Validate`.
- `spec.template.spec` with the same create validation used for
  `StackitMachineSpec`, but with field paths rooted at
  `spec.template.spec`.
- `spec.template.spec.providerID` should normally be forbidden in a template.
  Provider IDs are generated per machine by the controller.

Validate updates:

- `spec.template.spec` must be immutable.
- Allow `spec.template.metadata.labels` and annotations to change if valid.

### 6. Cross-Resource Checks

Be conservative. Admission webhooks should not call STACKIT APIs and should not
require external cloud resources to exist.

For Kubernetes resources:

- Do not require credential Secrets or bastion cloud-init ConfigMaps/Secrets to
  exist on create. This breaks declarative apply ordering. The controller should
  continue reporting missing references via conditions.
- It is acceptable to validate reference shape and namespace semantics.
- If you add a live Kubernetes lookup, justify it in comments and tests, and
  ensure it does not break ClusterClass dry-run, server-side apply, or GitOps
  ordering.

### 7. Tests

Add webhook unit tests close to the webhook package, for example:

```text
internal/webhook/v1alpha1/stackitcluster_webhook_test.go
internal/webhook/v1alpha1/stackitmachine_webhook_test.go
internal/webhook/v1alpha1/stackitclustertemplate_webhook_test.go
internal/webhook/v1alpha1/stackitmachinetemplate_webhook_test.go
```

Test defaulting directly by calling `Default`.

Test validation directly by calling:

- `ValidateCreate`
- `ValidateUpdate`
- `ValidateDelete`

Required test coverage:

- valid minimal `StackitCluster`
- invalid `StackitCluster` without credentials Secret name
- disabled API server load balancer without control-plane endpoint
- enabled bastion missing required fields
- valid bastion with `cloudInitRef`
- supported bastion `cloudInitRef` update
- rejected immutable `StackitCluster` field update
- valid minimal `StackitMachine`
- rejected invalid providerID
- allowed nil-to-valid-providerID update
- rejected providerID change
- rejected immutable `StackitMachine` creation-field update
- valid template metadata labels/annotations
- invalid template metadata labels/annotations
- rejected `StackitClusterTemplate.spec.template.spec` update
- rejected `StackitMachineTemplate.spec.template.spec` update
- rejected `StackitMachineTemplate.spec.template.spec.providerID`

Also add envtest coverage if direct webhook unit tests do not verify the
generated webhook server path. The existing `internal/controller/suite_test.go`
uses envtest but does not start webhooks today. Either add a dedicated webhook
envtest suite or extend the test environment carefully so it starts the webhook
server and validates admission through the API server.

### 8. Correctness Validation

Add an explicit validation step to prove the webhooks work in three layers:
direct Go tests, generated manifest checks, and live Kubernetes admission.

#### Direct Go Validation

The webhook tests must assert both the error type and the field path. Do not
only assert that "an error occurred".

For invalid requests, assert:

- the returned error is a Kubernetes invalid-object error when appropriate
- the `field.ErrorList` contains the expected path, for example:
  - `spec.credentialsSecretRef.name`
  - `spec.bastion.allowedCIDRs`
  - `spec.template.metadata.labels`
  - `spec.template.spec`
  - `spec.template.spec.providerID`
- the error message is clear enough for a user to fix the manifest without
  reading provider code

For defaulting, assert:

- the first `Default` call sets the expected field
- a second `Default` call does not change the object again
- template defaulting and non-template defaulting produce equivalent nested
  specs where expected

For update validation, assert:

- allowed changes pass, for example metadata-only template changes and
  controller-owned nil-to-valid `providerID` updates
- rejected changes fail at the exact field path that changed
- immutable template spec changes include guidance to create a new template

#### Generated Manifest Validation

After `make manifests`, inspect the generated files and make the test or final
report prove:

- every STACKIT API type with a validator has a corresponding
  `ValidatingWebhookConfiguration` rule
- every STACKIT API type with a defaulter has a corresponding
  `MutatingWebhookConfiguration` rule
- webhook rules use `failurePolicy: Fail`
- webhook rules use `sideEffects: None`
- webhook names are unique and include the STACKIT API group
- the manager Deployment has the webhook server port, certificate volume, and
  mount expected by the generated patch
- the default kustomization actually includes webhook resources and patches
- `make clusterctl-release` embeds webhook configurations in
  `infrastructure-components.yaml`

Suggested static checks:

```sh
rg -n "ValidatingWebhookConfiguration|MutatingWebhookConfiguration" config dist/clusterctl
rg -n "stackitclusters|stackitmachines|stackitclustertemplates|stackitmachinetemplates" config/webhook dist/clusterctl
rg -n "failurePolicy: Fail|sideEffects: None" config/webhook dist/clusterctl
rg -n "webhook-server|9443|cert" config/default config/manager dist/clusterctl
```

#### Live Admission Validation

Run a live admission smoke test against an isolated kind cluster. This catches
the class of failures direct unit tests miss: broken cert wiring, missing
kustomize resources, webhook service name mistakes, and admission paths that
are not registered.

The live validation must:

1. Install CRDs and deploy the manager with webhook configuration enabled.
2. Wait for the manager Deployment to become available.
3. Confirm validating and mutating webhook configurations exist.
4. Apply a valid object and confirm it is accepted.
5. Apply invalid objects and confirm rejection happens through the webhook with
   the expected field path.
6. Apply an object missing a defaulted field, then read it back and confirm the
   default was persisted.
7. Attempt an invalid update, for example changing
   `StackitMachineTemplate.spec.template.spec.machineType`, and confirm the API
   server rejects the update.

The invalid-object smoke tests must include at least:

- `StackitMachineTemplate` with `spec.template.spec.providerID`
- `StackitMachineTemplate` update changing `spec.template.spec.machineType`
- `StackitCluster` with bastion enabled but no `allowedCIDRs`

Do not accept a result where the CRD schema rejects the object before the
webhook. Pick cases that specifically exercise webhook-only validation, such as
template immutability or providerID-in-template rejection.

#### Workload Validation

If webhook changes can affect normal cluster creation, run a real workload e2e:

```sh
make test-e2e-workload-noderef
```

If webhook changes affect ClusterClass or template behaviour, also run:

```sh
make test-e2e-workload-topology
```

The e2e result is only acceptable if:

- the workload cluster reaches Ready
- `cloud-provider-stackit` rolls out
- CNI installs successfully
- Nodes have STACKIT provider IDs
- Nodes do not retain the external cloud-provider taint
- cleanup confirms no tagged STACKIT resources remain

### 9. Manifests And Packaging

After editing webhook code/markers or API markers, run:

```sh
make manifests
make generate
```

Ensure generated output includes:

- `config/webhook/manifests.yaml`
- `config/webhook/service.yaml`
- `config/default/manager_webhook_patch.yaml`
- validating webhook configuration entries for all four resources
- mutating webhook configuration entries for resources with defaulters
- CA injection wiring if cert-manager is used
- manager Deployment exposes/mounts webhook certificates as required

Ensure `make clusterctl-release` includes the webhook configuration in
`dist/clusterctl/.../infrastructure-components.yaml`.

Do not manually edit:

- `config/crd/bases/*.yaml`
- `config/rbac/role.yaml`
- generated webhook manifests
- `api/v1alpha1/zz_generated.deepcopy.go`
- `PROJECT`

Regenerate them through Kubebuilder/controller-tools only.

### 10. Validation Commands

Run all of the following before finishing:

```sh
make manifests
make generate
make test
make clusterctl-release
```

Then inspect generated manifests:

```sh
rg -n "ValidatingWebhookConfiguration|MutatingWebhookConfiguration|stackitclusters|stackitmachines|stackitclustertemplates|stackitmachinetemplates" config dist/clusterctl
rg -n "aggregate-to-manager" config/rbac/capi_manager_aggregation_role.yaml dist/clusterctl
```

Create an isolated kind cluster or reuse the dedicated e2e kind cluster, then
verify install-time admission works:

```sh
kind get clusters
make install
make deploy IMG=example.com/cluster-api-provider-stackit:webhook-test
kubectl get validatingwebhookconfigurations
kubectl get mutatingwebhookconfigurations
```

Apply at least one invalid object and confirm the API server returns a clear
field error from the webhook. Do not rely only on CRD schema rejection.

Example scenarios:

- `StackitMachineTemplate` with `spec.template.spec.providerID`
- `StackitMachineTemplate` update changing `spec.template.spec.machineType`
- `StackitCluster` with bastion enabled but no `allowedCIDRs`

If webhook deployment requires cert-manager, validate cert-manager is installed
or document the local install prerequisite clearly.

Run at least one real workload e2e if webhook behaviour can affect normal
cluster creation:

```sh
make test-e2e-workload-noderef
```

If webhook changes affect ClusterClass/template behaviour, also run:

```sh
make test-e2e-workload-topology
```

The final report must include:

- JJ change ID
- files changed
- which validation/defaulting rules were implemented
- generated manifest checks performed
- unit/envtest results
- e2e test ID if billable e2e was run
- any rules intentionally deferred, with reasons

## Non-Goals

- Do not add API version conversion webhooks unless you introduce a second API
  version.
- Do not call STACKIT APIs from admission webhooks.
- Do not use webhooks as a substitute for controller conditions. Runtime
  availability and eventual consistency failures still belong in reconciliation
  status.
- Do not remove existing CRD validation. Webhooks should complement CRD schema
  validation.

# Instructions: Complete ClusterClass Support

## Goal

Complete ClusterClass support for `cluster-api-provider-stackit`.

`StackitClusterTemplate` and `StackitMachineTemplate` already exist, but their
`spec.template` resources do not yet expose template metadata. Add support for:

```yaml
spec:
  template:
    metadata:
      labels:
        example: value
      annotations:
        example: value
    spec:
      ...
```

Cluster API topology must then be able to create the generated
`StackitCluster` and `StackitMachine` objects with those labels and
annotations copied onto the generated infrastructure objects.

Full create, ready, and delete e2e coverage for topology clusters is required.
The implementation is not complete until the topology e2e test passes against
real STACKIT resources and verifies there are no leaked STACKIT resources after
deletion.

## Required VCS Workflow

This repository uses Jujutsu. Do not use raw Git for normal VCS operations.

Before editing:

```sh
jj status
```

If the working copy is not an empty change, create a dedicated change:

```sh
jj new -m "Complete ClusterClass template metadata support"
```

If the working copy is empty but undescribed, describe it:

```sh
jj desc -m "Complete ClusterClass template metadata support"
```

After each meaningful implementation step, check:

```sh
jj status
jj diff --git
```

Use `jj diff --git`, not plain `jj diff`, so the output is normal unified diff
format. Do not run destructive commands such as `git reset --hard`,
`git checkout --`, or `jj restore` unless explicitly asked.

Keep unrelated work out of this change. If you discover unrelated local changes,
do not revert them; either work around them or create a separate JJ change.

## What ClusterClass Template Metadata Means

ClusterClass topology generates concrete objects from templates. The
infrastructure cluster template produces a `StackitCluster`; machine
infrastructure templates produce `StackitMachine` objects for control plane and
worker Machines.

Template metadata is the labels and annotations stored under
`spec.template.metadata` on a template object. Cluster API copies that metadata
onto the generated object. This is important for topology users because labels
and annotations are often used for policy, ownership, selection, cost tracking,
observability, and automation.

Do not use full Kubernetes `metav1.ObjectMeta` for template metadata. Template
metadata should only carry labels and annotations. Follow the local Cluster API
v1beta2 API pattern exactly.

## What RBAC Aggregation Means

Kubernetes `ClusterRole` aggregation lets one `ClusterRole` collect rules from
other `ClusterRole` objects using label selectors. The collecting role has an
`aggregationRule`; Kubernetes continuously updates its effective `rules` from
all matching roles.

Cluster API core uses this for its manager permissions. In the local Cluster API
checkout, `config/rbac/aggregated_role.yaml` defines an
`aggregated-manager-role` that selects roles with:

```yaml
cluster.x-k8s.io/aggregate-to-manager: "true"
```

Provider roles with that label grant Cluster API core access to provider custom
resources. This is required for ClusterClass/topology because CAPI core creates
resources from infrastructure templates, updates metadata, and manages owner
references on provider resources.

This repository should already contain CAPI manager aggregation RBAC for
`StackitCluster`, `StackitClusterTemplate`, `StackitMachine`, and
`StackitMachineTemplate`. Keep it intact. The template metadata feature is a
separate API/schema change, but ClusterClass support is incomplete if CAPI core
does not also have the aggregated RBAC it needs to operate on these resources.

## Primary References

Use the local Cluster API checkout as the main API reference:

```text
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/cluster-api
```

Read these local Cluster API files before implementing:

- `api/bootstrap/kubeadm/v1beta2/kubeadmconfigtemplate_types.go`
- `api/controlplane/kubeadm/v1beta2/kubeadmcontrolplanetemplate_types.go`
- `api/core/v1beta2/cluster_types.go`
- `api/core/v1beta2/clusterclass_types.go`
- `config/rbac/aggregated_role.yaml`
- `config/default/manager_role_aggregation_patch.yaml`
- `docs/book/src/developer/providers/contracts/infra-cluster.md`
- `docs/book/src/developer/providers/contracts/infra-machine.md`

The most important API pattern to mirror is:

```go
// metadata is the standard object's metadata.
// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
// +optional
ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty,omitzero"`
```

where `clusterv1` is:

```go
clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
```

Also read the provider files before editing:

- `api/v1alpha1/stackitclustertemplate_types.go`
- `api/v1alpha1/stackitmachinetemplate_types.go`
- `api/v1alpha1/stackitcluster_types.go`
- `api/v1alpha1/stackitmachine_types.go`
- `templates/clusterclass.yaml`
- `templates/cluster-template-topology.yaml`
- `test/e2e/e2e_test.go`
- `config/crd/bases/infrastructure.cluster.x-k8s.io_stackitclustertemplates.yaml`
- `config/crd/bases/infrastructure.cluster.x-k8s.io_stackitmachinetemplates.yaml`
- `config/rbac/capi_manager_aggregation_role.yaml`

## Implementation Plan

1. Add template metadata fields to the API types.

   In `api/v1alpha1/stackitclustertemplate_types.go`, update
   `StackitClusterTemplateResource` to include CAPI template metadata:

   ```go
   // metadata is the standard object's metadata.
   // More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
   // +optional
   ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty,omitzero"`
   ```

   Keep `Spec StackitClusterSpec` present and required unless careful research
   proves the local API should be loosened. This provider currently requires a
   template spec, and ClusterClass needs it.

   In `api/v1alpha1/stackitmachinetemplate_types.go`, update
   `StackitMachineTemplateResource` the same way.

   Add the required import:

   ```go
   clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
   ```

   Do not use `metav1.ObjectMeta` for `spec.template.metadata`. Full
   `metav1.ObjectMeta` would expose fields like `name`, `namespace`,
   `ownerReferences`, and `managedFields`, which are not valid template
   metadata semantics.

2. Regenerate generated artifacts.

   After editing API types, run:

   ```sh
   make manifests
   make generate
   ```

   Do not manually edit these generated files:

   - `config/crd/bases/*.yaml`
   - `api/v1alpha1/zz_generated.deepcopy.go`
   - `config/rbac/role.yaml`

   Inspect the generated CRDs and verify that both template CRDs now expose:

   - `spec.template.metadata.labels`
   - `spec.template.metadata.annotations`

   The schema should not expose arbitrary full object metadata fields under
   `spec.template.metadata`.

3. Update ClusterClass fixtures/templates.

   Update `templates/clusterclass.yaml` so the STACKIT infrastructure templates
   demonstrate template metadata.

   Add deterministic labels and annotations to:

   - `StackitClusterTemplate.spec.template.metadata`
   - the control plane `StackitMachineTemplate.spec.template.metadata`
   - the worker `StackitMachineTemplate.spec.template.metadata`

   Use labels and annotations that are safe for tests, for example:

   ```yaml
   metadata:
     labels:
       cluster-api-provider-stackit/template-metadata: cluster
     annotations:
       cluster-api-provider-stackit/template-metadata: cluster
   ```

   For machine templates, use distinct values such as `control-plane` and
   `worker` so e2e checks can prove the correct metadata came from the correct
   template.

   If the repository has docs or sample ClusterClass manifests that should stay
   in sync, update those too. Keep changes focused on ClusterClass support.

4. Strengthen topology e2e coverage.

   The existing e2e suite already has a topology workload path gated by:

   ```text
   STACKIT_E2E_TOPOLOGY_WORKLOAD=true
   ```

   Extend that test instead of adding a parallel incomplete test.

   The topology e2e must prove all of the following:

   - the ClusterClass fixture applies successfully
   - the topology `Cluster` fixture applies successfully
   - the generated workload cluster reaches ready state
   - the expected control plane and worker Machines reach Ready
   - the workload Kubernetes Nodes reach Ready
   - the generated `StackitCluster` has labels and annotations from
     `StackitClusterTemplate.spec.template.metadata`
   - generated control plane `StackitMachine` objects have labels and
     annotations from the control plane `StackitMachineTemplate`
   - generated worker `StackitMachine` objects have labels and annotations from
     the worker `StackitMachineTemplate`
   - deleting the topology `Cluster` deletes generated CAPI and STACKIT
     infrastructure objects
   - no STACKIT servers or API server load balancers tagged with the e2e test ID
     remain after deletion

   Be precise when selecting generated objects. Use normal Cluster API labels
   such as:

   - `cluster.x-k8s.io/cluster-name=<clusterName>`
   - `cluster.x-k8s.io/control-plane`
   - `cluster.x-k8s.io/deployment-name=<machineDeploymentName>`

   Do not assume object names unless the test already controls them. Prefer
   label selectors and JSON output parsed by Go test code.

5. Add helper functions only if they make the e2e clearer.

   If metadata assertions would duplicate a lot of code, add small focused
   helpers in `test/e2e/e2e_test.go`. Keep them local to the e2e file unless an
   existing helper package already owns this kind of assertion.

   Good helper shapes:

   - get one object as JSON and assert a label/annotation value
   - list objects by selector and assert all selected objects have a
     label/annotation value
   - wait until generated infra objects exist before checking metadata

   Avoid shell pipelines in Go tests. Use `kubectl -o json` with Go JSON
   parsing, or the existing e2e helper conventions in this repository.

6. Re-check CAPI manager RBAC aggregation.

   Confirm that the provider still ships a `ClusterRole` labeled:

   ```yaml
   cluster.x-k8s.io/aggregate-to-manager: "true"
   ```

   and that it grants CAPI core create/delete/get/list/patch/update/watch on:

   - `stackitclusters`
   - `stackitclustertemplates`
   - `stackitmachines`
   - `stackitmachinetemplates`

   Do not remove this role while working on template metadata. It is required
   for ClusterClass-created provider resources.

7. Add explicit correctness validation.

   Do not rely on "the manifests generated" as proof that ClusterClass support
   is correct. Add validation that proves the behavior users depend on.

   The validation must cover these cases:

   - API schema accepts `spec.template.metadata.labels` and
     `spec.template.metadata.annotations` on `StackitClusterTemplate`
   - API schema accepts `spec.template.metadata.labels` and
     `spec.template.metadata.annotations` on `StackitMachineTemplate`
   - API schema does not accidentally expose full `metav1.ObjectMeta` fields
     under `spec.template.metadata`, such as `name`, `namespace`,
     `ownerReferences`, or `managedFields`
   - CAPI core manager aggregation RBAC is present in the rendered install
     manifests
   - topology reconciliation copies cluster template metadata onto the generated
     `StackitCluster`
   - topology reconciliation copies control plane machine template metadata onto
     generated control plane `StackitMachine` objects
   - topology reconciliation copies worker machine template metadata onto
     generated worker `StackitMachine` objects
   - deletion of the topology `Cluster` removes generated CAPI objects,
     generated STACKIT provider objects, and external STACKIT resources

   Prefer automated validation in tests. The e2e test must include metadata
   propagation assertions, because schema-only validation cannot prove CAPI
   topology actually copies the metadata to generated infrastructure objects.

## Validation

Run local generation and unit validation:

```sh
make manifests
make generate
make test
```

Inspect generated schema:

```sh
rg -n "metadata:|labels:|annotations:" config/crd/bases/infrastructure.cluster.x-k8s.io_stackitclustertemplates.yaml
rg -n "metadata:|labels:|annotations:" config/crd/bases/infrastructure.cluster.x-k8s.io_stackitmachinetemplates.yaml
```

Also inspect that the generated `spec.template.metadata` schema contains only
template metadata fields. This check must fail the implementation if full object
metadata leaked into the template schema:

```sh
rg -n "ownerReferences|managedFields|generateName|resourceVersion|uid" \
  config/crd/bases/infrastructure.cluster.x-k8s.io_stackitclustertemplates.yaml \
  config/crd/bases/infrastructure.cluster.x-k8s.io_stackitmachinetemplates.yaml
```

If this command matches fields inside `spec.template.metadata`, the
implementation used the wrong metadata type. Matches outside the template
metadata schema must be inspected carefully before accepting them.

Render the default installation to catch kustomize errors:

```sh
kubectl kustomize config/default
```

Verify the rendered install still contains CAPI manager aggregation RBAC:

```sh
kubectl kustomize config/default | rg -n "aggregate-to-manager|stackitclusters|stackitclustertemplates|stackitmachines|stackitmachinetemplates"
```

Render the topology cluster template with `clusterctl` using the repository's
normal environment. At minimum, verify the template renders without errors:

```sh
clusterctl generate cluster "$CLUSTER_NAME" \
  --from templates/cluster-template-topology.yaml \
  --target-namespace "$NAMESPACE"
```

For schema validation against a real API server, install the generated CRDs into
an isolated Kind management cluster and apply minimal template objects that
include labels and annotations under `spec.template.metadata`.

Use server-side validation, not only client rendering:

```sh
kubectl apply --server-side --dry-run=server -f <minimal-stackit-cluster-template-with-template-metadata.yaml>
kubectl apply --server-side --dry-run=server -f <minimal-stackit-machine-template-with-template-metadata.yaml>
```

Then try a negative validation fixture with an invalid full object metadata
field under `spec.template.metadata`. It should be rejected:

```sh
kubectl apply --server-side --dry-run=server -f <invalid-template-metadata-with-ownerReferences.yaml>
```

Run the real topology e2e test against an isolated Kind management cluster and
real STACKIT resources:

```sh
make test-e2e-workload-topology
```

If running the test directly, preserve the same focus and timeout as the
Makefile target:

```sh
env STACKIT_E2E_TOPOLOGY_WORKLOAD=true \
  KIND_CLUSTER=cluster-api-provider-stackit-test-e2e \
  go test -timeout=90m -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='topology.*workload' --ginkgo.timeout=90m
```

The e2e test must use an isolated Kind cluster, not a real dev or production
management cluster.

During the e2e run, capture evidence for the metadata checks. The test itself
should assert these values, but the final report should also include equivalent
manual inspection commands or summarized output for:

```sh
kubectl get stackitcluster -n "$NAMESPACE" -l cluster.x-k8s.io/cluster-name="$CLUSTER_NAME" -o json
kubectl get stackitmachine -n "$NAMESPACE" -l cluster.x-k8s.io/cluster-name="$CLUSTER_NAME" -o json
```

Do not claim the feature is complete if the real topology e2e was skipped. If
STACKIT credentials or quota are unavailable, report that as a blocker and leave
the change clearly marked as unverified.

## Completion Criteria

The change is complete only when:

- `StackitClusterTemplate.spec.template.metadata.labels` and `.annotations`
  are accepted by the CRD schema
- `StackitMachineTemplate.spec.template.metadata.labels` and `.annotations`
  are accepted by the CRD schema
- ClusterClass topology creates generated `StackitCluster` and `StackitMachine`
  resources with the expected propagated metadata
- the topology workload cluster reaches Ready with real STACKIT resources
- deleting the topology Cluster removes generated CAPI objects and provider
  infrastructure objects
- the e2e cleanup check finds no STACKIT servers or API server load balancers
  tagged for the test ID
- `make manifests`, `make generate`, `make test`, and the topology e2e test
  pass
- `jj diff --git` contains only the intended API, generated manifest, template,
  test, and documentation changes

## Final Report

When finished, report:

- which API types changed
- which generated files changed
- which ClusterClass fixtures/templates changed
- what metadata propagation assertions were added
- exact validation commands run and whether they passed
- the topology e2e cluster name and `STACKIT_E2E_TEST_ID`
- confirmation that no tagged STACKIT resources remained after deletion
- the final `jj status` summary

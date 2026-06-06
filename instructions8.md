# Instructions: Provide a Tested Addon Flow for cloud-provider-stackit and CNI

## Goal

Provide a tested addon flow for:

- `cloud-provider-stackit`
- workload-cluster CNI installation

The final user-facing result must be one of these:

1. a `clusterctl`-friendly addon template flow that includes all required
   objects and can be rendered/applied predictably, or
2. a documented, tested post-create flow that reliably installs the CNI, clears
   the external cloud-provider node taint, and aligns provider IDs.

Prefer the smallest robust change. This repository already has:

- `templates/addons/cloud-provider-stackit.yaml`
- `ClusterResourceSet` wiring in `templates/cluster-template.yaml`
- `ClusterResourceSet` wiring in `templates/cluster-template-topology.yaml`
- `hack/install-workload-cni.sh`
- `make install-workload-cni`
- e2e checks for Ready Nodes and providerID alignment

The implementation should harden and document this existing flow rather than
inventing a second addon system unless careful research proves the current
shape is not clusterctl-friendly enough.

## Required VCS Workflow

This repository uses Jujutsu. Do not use raw Git for normal VCS operations.

Before editing:

```sh
jj status
```

If the working copy is not an empty change, create a dedicated change:

```sh
jj new -m "Add tested workload addon flow"
```

If the working copy is empty but undescribed, describe it:

```sh
jj desc -m "Add tested workload addon flow"
```

After each meaningful step, check:

```sh
jj status
jj diff --git
```

Use `jj diff --git`, not plain `jj diff`, so the output is normal unified diff
format. Keep unrelated changes out of this change. Do not use destructive
commands such as `git reset --hard`, `git checkout --`, or `jj restore` unless
explicitly asked.

## What RBAC Aggregation Means

Kubernetes `ClusterRole` aggregation lets one `ClusterRole` collect rules from
other `ClusterRole` objects using label selectors. The collecting role has an
`aggregationRule`; Kubernetes continuously updates its effective `rules` from
all matching roles.

Cluster API core uses this for its manager permissions. In the local Cluster API
checkout, `config/rbac/aggregated_role.yaml` defines an
`aggregated-manager-role` that selects roles labeled:

```yaml
cluster.x-k8s.io/aggregate-to-manager: "true"
```

Provider roles with that label grant CAPI core access to provider custom
resources. This matters for addon flow because CAPI core must be able to manage
ClusterClass/topology-generated infrastructure resources, owner references, and
metadata while addons such as `cloud-provider-stackit` reconcile workload Nodes.

Do not remove or weaken the provider's CAPI manager aggregation RBAC while
working on addon templates or docs. This feature is not primarily an RBAC
change, but reliable ClusterClass and addon behavior depends on CAPI core still
having the aggregated provider permissions it needs.

## Primary References

Read this codebase carefully before editing:

- `templates/addons/cloud-provider-stackit.yaml`
- `templates/addons/cilium-values.yaml`
- `templates/cluster-template.yaml`
- `templates/cluster-template-development.yaml`
- `templates/cluster-template-topology.yaml`
- `hack/install-workload-cni.sh`
- `Makefile`
- `docs/src/usage/cluster-template.md`
- `docs/src/usage/clusterclass.md`
- `docs/src/usage/cni.md`
- `docs/src/reference/variables.md`
- `docs/src/development/testing.md`
- `test/e2e/e2e_test.go`
- `pkg/cloud/providerid.go`
- `pkg/cloud/providerid_test.go`

Use the local Cluster API checkout as the main API reference:

```text
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/cluster-api
```

Read these files there:

- `docs/book/src/developer/providers/contracts/clusterctl.md`
- `api/addons/v1beta2/clusterresourceset_types.go`
- `api/addons/v1beta2/clusterresourcesetbinding_types.go`
- `docs/proposals/20200220-cluster-resource-set.md`

Important CAPI constraints to preserve:

- `clusterctl` cluster templates should include all required objects. Avoid
  templates that depend on unrendered external Secrets or ConfigMaps unless the
  docs explicitly explain that requirement and the flow is tested.
- `ClusterResourceSet` resources are same-namespace `Secret` or `ConfigMap`
  objects.
- `ClusterResourceSet` Secrets must use type
  `addons.cluster.x-k8s.io/resource-set`.
- `ClusterResourceSet` is a simple static addon mechanism. It is useful for
  bootstrap addons, but it is not a full addon lifecycle manager for upgrades,
  rollback, or drift reconciliation.

## Desired User Flow

Design for a user who creates a cluster from this repository's templates:

1. Render a cluster with `clusterctl generate cluster`.
2. Apply the rendered cluster manifest.
3. Wait until the workload API is reachable.
4. Install the CNI with a documented command.
5. Verify that:
   - `cloud-provider-stackit` is deployed in the workload cluster
   - the CNI is rolled out
   - Nodes are Ready
   - `node.cloudprovider.kubernetes.io/uninitialized` is gone
   - `Machine.spec.providerID`, `StackitMachine.status.providerID`, and
     `Node.spec.providerID` match

Keep production guidance honest: production users may manage CNI through Helm,
GitOps, or an addon provider. The repository must still provide one tested
development/default flow.

## Implementation Plan

1. Decide and document the supported addon model.

   Recommended model for this repository:

   - Keep `cloud-provider-stackit` embedded in the cluster templates through a
     `ClusterResourceSet` and resource-set `Secret`.
   - Keep CNI installation as an explicit post-create step using
     `make install-workload-cni`, backed by `hack/install-workload-cni.sh`.
   - Document Cilium as the default tested CNI and Calico/custom manifests as
     supported helper paths.

   If you decide instead to ship a standalone `clusterctl` addon template, it
   must include all objects needed by `clusterctl` and must not rely on hidden
   external state. It must be copied into `dist/clusterctl` by
   `make clusterctl-release` and covered by tests/docs.

2. Harden `cloud-provider-stackit` addon rendering.

   Inspect `templates/addons/cloud-provider-stackit.yaml` and every place it is
   embedded into cluster templates.

   Confirm the rendered workload addon includes:

   - `Namespace/kube-system`
   - `ServiceAccount/stackit-cloud-controller-manager`
   - `ConfigMap/stackit-cloud-config`
   - `Secret/stackit-cloud-secret`
   - `ClusterRole/stackit-cloud-controller-manager`
   - `ClusterRoleBinding/stackit-cloud-controller-manager`
   - `Deployment/stackit-cloud-controller-manager`

   Confirm it sets:

   - `--cloud-provider=stackit`
   - `--controllers=cloud-node-controller,cloud-node-lifecycle-controller,service-lb-controller`
   - `--cluster-name=${CLUSTER_NAME}`
   - `STACKIT_SERVICE_ACCOUNT_KEY_PATH=/etc/serviceaccount/sa_key.json`
   - toleration for `node.cloudprovider.kubernetes.io/uninitialized`

   Confirm the cluster templates configure kubeadm for external cloud provider:

   - kubelet `cloud-provider=external`
   - controller-manager `cloud-provider=external`

   Confirm `STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE` is validated against the
   workload Kubernetes minor. The existing e2e helper
   `cloudProviderStackitImageForKubernetesVersion` does this; docs or scripts
   should point users at `hack/validate-stackit-versions.sh`.

3. Harden the CNI helper path.

   Inspect `hack/install-workload-cni.sh` and make it safe and predictable:

   - require `WORKLOAD_KUBECONFIG` or `KUBECONFIG`
   - default to Cilium
   - use `templates/addons/cilium-values.yaml` for the default Cilium values
   - support Calico through `STACKIT_WORKLOAD_CNI=calico`
   - support custom manifests through `CNI_MANIFEST`
   - wait for the CNI rollout before returning

   If the script currently exits immediately for custom manifests without
   waiting, decide whether to document that explicitly or add a configurable
   wait. Do not imply custom manifests are fully verified unless the script
   actually waits for them.

4. Add explicit verification commands.

   Add or update docs so a user can verify addon success after creating a
   cluster:

   ```sh
   clusterctl get kubeconfig "${CLUSTER_NAME}" \
     --namespace "${NAMESPACE}" \
     > "${CLUSTER_NAME}.kubeconfig"

   make install-workload-cni \
     WORKLOAD_KUBECONFIG="${CLUSTER_NAME}.kubeconfig"

   kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" \
     -n kube-system rollout status deployment/stackit-cloud-controller-manager --timeout=5m

   kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" \
     get nodes -o custom-columns=NAME:.metadata.name,READY:.status.conditions[-1].status,PROVIDER_ID:.spec.providerID,TAINTS:.spec.taints
   ```

   Prefer robust commands over clever JSONPath. If you need to assert no
   external cloud-provider taint remains, use `jq` if the docs already assume
   it or provide a clear `kubectl -o json` command with a small explanation.

5. Add a dedicated docs chapter or substantially improve the existing CNI docs.

   The docs should explain:

   - `cloud-provider-stackit` is installed by the cluster templates through
     CAPI `ClusterResourceSet`.
   - The CNI is deliberately not installed by the cluster template.
   - Nodes will generally not become Ready until a CNI is installed.
   - With `cloud-provider=external`, Kubernetes adds
     `node.cloudprovider.kubernetes.io/uninitialized` until the cloud
     controller initializes the Node.
   - `cloud-provider-stackit` is responsible for initializing Nodes and setting
     `Node.spec.providerID`.
   - The infrastructure provider writes `StackitMachine.spec.providerID` and
     `StackitMachine.status.providerID`; CAPI then aligns the owning Machine.
   - The expected providerID format is `stackit://<server-id>`.
   - The tested default flow is:
     1. render/apply cluster
     2. retrieve kubeconfig
     3. run `make install-workload-cni`
     4. wait for `stackit-cloud-controller-manager`
     5. verify Ready Nodes, no external cloud-provider taint, and providerID
        alignment

   Suggested doc location:

   ```text
   docs/src/usage/addons.md
   ```

   If adding this chapter, wire it into:

   ```text
   docs/src/SUMMARY.md
   docs/src/usage/index.md
   ```

   Then either keep `docs/src/usage/cni.md` as the CNI-specific subpage or
   merge it into the addon chapter. Avoid duplicated contradictory instructions.

6. Strengthen e2e validation.

   The e2e path already installs CNI, waits for `cloud-provider-stackit`,
   verifies Ready Nodes, and verifies providerID alignment. Make the assertions
   explicit enough that the addon flow cannot regress unnoticed.

   Add or verify assertions for:

   - `deployment/stackit-cloud-controller-manager` rolls out in workload
     `kube-system`
   - selected CNI rolls out
   - each Node is Ready
   - each Node has `spec.providerID` with prefix `stackit://`
   - no Node has the taint key
     `node.cloudprovider.kubernetes.io/uninitialized`
   - `Machine.spec.providerID` equals the matching
     `StackitMachine.status.providerID`
   - `Node.spec.providerID` equals the matching
     `Machine.spec.providerID`

   Prefer adding a direct check in `expectWorkloadNodesReady` for the taint.
   That helper already checks Ready Nodes and providerID and is used by the real
   workload e2e paths.

   If the e2e supports both Cilium and Calico, ensure the default Cilium path is
   the one documented as tested. Calico/custom-manifest paths can be documented
   as helper options unless they have dedicated e2e coverage.

7. Keep release packaging consistent.

   `make clusterctl-release` currently copies `templates/addons/*.yaml` into:

   ```text
   dist/clusterctl/infrastructure-stackit/<version>/addons/
   ```

   Verify that this still happens. If you add a new addon template, ensure it is
   included in the generated release assets.

   Run:

   ```sh
   make clusterctl-release
   ```

   Then inspect the generated release directory for:

   - `cluster-template.yaml`
   - `cluster-template-topology.yaml`
   - `addons/cloud-provider-stackit.yaml`
   - `addons/cilium-values.yaml`
   - any new addon docs/templates you add

8. Add explicit correctness validation.

   Do not rely on "the e2e passed" or "the docs build passed" as the only proof
   of correctness. Add validation that directly proves the addon flow satisfies
   the user-facing contract.

   The validation must cover:

   - rendered classic and topology cluster templates include exactly one
     `ClusterResourceSet` for `cloud-provider-stackit`
   - each `ClusterResourceSet` references a same-namespace resource-set
     `Secret` of type `addons.cluster.x-k8s.io/resource-set`
   - the resource-set Secret embeds a `Deployment` named
     `stackit-cloud-controller-manager`
   - the embedded `cloud-provider-stackit` deployment uses the selected
     `STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE`
   - the selected `cloud-provider-stackit` image minor matches
     `KUBERNETES_VERSION`
   - the CNI helper installs the selected default CNI and waits for rollout
   - the workload cluster has Ready Nodes after CNI and CCM rollout
   - no workload Node has the taint key
     `node.cloudprovider.kubernetes.io/uninitialized`
   - every workload Node has `spec.providerID`
   - every workload Node providerID matches the owning CAPI Machine providerID
   - every CAPI Machine providerID matches the referenced `StackitMachine`
     providerID
   - release assets include all addon templates required by the documented flow

   Prefer automated checks in e2e tests for runtime behavior. For static
   template/release assertions, use focused unit tests, shell checks in the
   validation instructions, or a small script if that matches existing repo
   patterns. Manual inspection is acceptable only as a supplemental check; it
   must not be the only validation for taint removal or providerID alignment.

## Validation

Run local validation:

```sh
make test
make clusterctl-release
make -C docs build
```

Render the classic and topology templates with realistic environment variables
and save them for inspection:

```sh
clusterctl generate cluster "${CLUSTER_NAME}" \
  --from templates/cluster-template.yaml \
  --target-namespace "${NAMESPACE}" \
  --kubernetes-version "${KUBERNETES_VERSION}" \
  --control-plane-machine-count "${CONTROL_PLANE_MACHINE_COUNT}" \
  --worker-machine-count "${WORKER_MACHINE_COUNT}" \
  > /tmp/stackit-classic-addon-flow.yaml

clusterctl generate cluster "${CLUSTER_NAME}" \
  --from templates/cluster-template-topology.yaml \
  --target-namespace "${NAMESPACE}" \
  --kubernetes-version "${KUBERNETES_VERSION}" \
  --control-plane-machine-count "${CONTROL_PLANE_MACHINE_COUNT}" \
  --worker-machine-count "${WORKER_MACHINE_COUNT}" \
  > /tmp/stackit-topology-addon-flow.yaml
```

Inspect the rendered addon resources. Use a structured YAML tool such as `yq`
if available; otherwise inspect the rendered manifests carefully. At minimum,
confirm both files contain:

```sh
rg -n "kind: ClusterResourceSet|name: .*cloud-provider-stackit|type: addons.cluster.x-k8s.io/resource-set|stackit-cloud-controller-manager|node.cloudprovider.kubernetes.io/uninitialized|\\$\\{" \
  /tmp/stackit-classic-addon-flow.yaml \
  /tmp/stackit-topology-addon-flow.yaml
```

The rendered manifests must not contain unresolved `${...}` variables. If the
`rg` command matches `${`, fix template substitution before running e2e.

Verify release assets include addon files:

```sh
test -f dist/clusterctl/infrastructure-stackit/v0.1.0/addons/cloud-provider-stackit.yaml
test -f dist/clusterctl/infrastructure-stackit/v0.1.0/addons/cilium-values.yaml
```

Run real e2e with billable STACKIT resources:

```sh
make test-e2e-workload-noderef
```

This is the minimum e2e path for addon validation because it creates a workload
cluster, installs CNI, waits for `cloud-provider-stackit`, waits for Ready
Nodes, and verifies providerID alignment. After implementing this change, this
e2e must also assert that no Node still has
`node.cloudprovider.kubernetes.io/uninitialized`.

If this change touches topology addon behavior, also run:

```sh
make test-e2e-workload-topology
```

Capture or summarize the e2e evidence for:

```sh
kubectl --kubeconfig <workload-kubeconfig> \
  -n kube-system rollout status deployment/stackit-cloud-controller-manager --timeout=5m

kubectl --kubeconfig <workload-kubeconfig> \
  get nodes -o json

kubectl get machine,stackitmachine -n <namespace> \
  -l cluster.x-k8s.io/cluster-name=<cluster-name> -o json
```

The final report must state whether the e2e assertions proved Ready Nodes,
providerID alignment, and absence of the external cloud-provider taint.

During e2e, capture the cluster name and `STACKIT_E2E_TEST_ID`. After deletion,
verify no tagged STACKIT resources remain through the existing e2e cleanup check
or by running:

```sh
make cleanup-stackit STACKIT_E2E_TEST_ID=<test-id>
```

Do not claim the addon flow is tested if the real e2e was skipped. If STACKIT
credentials, quota, `cilium`, or network access are unavailable, report that as
a blocker and leave the change clearly marked as not fully verified.

## Completion Criteria

The change is complete only when:

- docs describe one supported, tested addon flow end to end
- `cloud-provider-stackit` install behavior is clear for classic and topology
  templates
- CNI install behavior is clear and has a default tested path
- verification commands cover rollout, Ready Nodes, providerID alignment, and
  absence of the external cloud-provider taint
- e2e asserts no `node.cloudprovider.kubernetes.io/uninitialized` taint remains
- e2e asserts providerID alignment across `StackitMachine`, `Machine`, and
  `Node`
- `make test` passes
- `make -C docs build` passes
- `make clusterctl-release` includes addon assets
- billable e2e validates the default flow against real STACKIT resources

## Final Report

When finished, report:

- which addon flow was chosen and why
- which templates changed
- which docs changed
- which e2e assertions were added or confirmed
- exact validation commands run and whether they passed
- e2e cluster name and `STACKIT_E2E_TEST_ID`
- confirmation that no tagged STACKIT resources remained after e2e deletion
- final `jj status`

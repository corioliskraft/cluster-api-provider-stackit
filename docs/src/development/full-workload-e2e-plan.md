# Full Workload-Cluster E2E Plan

This document records the remaining work to turn the current STACKIT e2e suite
from infrastructure-lifecycle coverage into full workload-cluster coverage. It
is written for an implementation agent and should be updated as each milestone
lands.

## Current Validation

The assumptions below were checked against the current code and docs:

- `test/e2e/e2e_test.go` has independently gated scenarios for VM lifecycle,
  cluster create/delete, NodeRef, worker scale, and worker upgrade.
- The NodeRef scenario gated by `STACKIT_E2E_NODE_REF=true` uses the real
  kubeadm fixture from `renderStackitKubeadmClusterFixture`. It extracts the
  workload kubeconfig, installs a CNI, waits for `cloud-provider-stackit`, waits
  for two workload Nodes to become Ready, and checks CAPI Machine to workload
  Node providerID alignment.
- The scale scenario gated by `STACKIT_E2E_SCALE_WORKERS=true` still uses
  `renderStackitMachineDeploymentScaleFixture`. That fixture creates only a
  `MachineDeployment`, `StackitMachineTemplate`, and a static bootstrap Secret
  with shell data. It does not create a real `KubeadmControlPlane`,
  `KubeadmConfigTemplate`, workload kubeconfig, CNI, or workload Nodes.
- The upgrade scenario gated by `STACKIT_E2E_UPGRADE_WORKERS=true` also uses
  `renderStackitMachineDeploymentScaleFixture`. It patches the
  `MachineDeployment` version, waits for a second infrastructure VM, then
  manually deletes the old CAPI `Machine` to exercise cleanup. It does not
  prove automatic rollout completion based on workload Node readiness.
- Scale, upgrade, and topology workload e2e coverage now exist as
  independently gated, billable scenarios. Remaining gaps are tracked in the
  later milestones and roadmap.
- `templates/clusterclass.yaml` and `templates/cluster-template-topology.yaml`
  include topology variables, the development kubeadm package-install fallback,
  and `cloud-provider-stackit` addon wiring for Ready workload Nodes.

I did not run the billable real-cloud e2e suite for this analysis.

## Principles

- Keep e2e scenarios independently executable. Each scenario must keep its own
  environment gate, test ID, cleanup path, resource tags, and focused Ginkgo
  name.
- Prefer real kubeadm workload fixtures for readiness-sensitive behavior.
  Static bootstrap Secrets are useful for infrastructure lifecycle checks, but
  they cannot prove node readiness, drain/delete behavior, or rollout gates.
- Reuse the NodeRef helper path where possible: kubeconfig extraction, CNI
  installation, `cloud-provider-stackit` rollout, workload Node readiness, and
  providerID alignment already exist.
- Keep direct cloud leak cleanup in every real-cloud scenario. Every STACKIT
  resource created by a test must be discoverable by the e2e test ID tags.
- Use separate milestones instead of one large rewrite so each behavior can be
  reviewed and, when needed, executed against STACKIT independently.

## Milestone 1: Shared Real Workload Fixture Helpers

Goal: make scale, upgrade, and topology tests reuse the proven NodeRef
workload-cluster path without copying large YAML strings.

### Current Code To Refactor

- `renderStackitKubeadmClusterFixture` currently renders the complete real
  workload fixture as one large `fmt.Sprintf` block. It already contains the
  correct non-topology object set for Ready Nodes:
  - `Cluster`
  - `StackitCluster`
  - `KubeadmControlPlane`
  - control-plane `StackitMachineTemplate`
  - worker `MachineDeployment`
  - worker `StackitMachineTemplate`
  - worker `KubeadmConfigTemplate`
  - `cloud-provider-stackit` `ClusterResourceSet`
  - `cloud-provider-stackit` Secret
- `renderStackitMachineDeploymentScaleFixture` is the infra-only fixture used
  by current scale and upgrade tests. It creates a static bootstrap Secret
  instead of CABPK kubeadm bootstrap objects, so it must not be reused for full
  workload readiness tests.
- The NodeRef scenario already contains the common runtime sequence that scale,
  upgrade, and topology tests need:
  - wait for `StackitCluster.status.ready=true`
  - wait for the control-plane `StackitMachine`
  - extract workload kubeconfig from `<cluster-name>-kubeconfig`
  - install CNI with `installWorkloadCNI`
  - wait for `cloud-provider-stackit` with `waitForWorkloadCCMRollout`
  - wait for Ready workload Nodes with `expectWorkloadNodesReady`
  - assert Machine/Node providerID alignment with
    `expectProviderIDNodeRefAlignment`

### Proposed Helper Types

Add small option structs near the e2e fixture rendering helpers. Keep them in
`test/e2e/e2e_test.go` first; move to a separate file only if the file becomes
hard to navigate.

```go
type kubeadmWorkloadFixtureOptions struct {
	ClusterName          string
	TestID               string
	Config               stackitVMConfig
	KubernetesVersion    string
	ServiceAccountJSON   []byte
	ControlPlaneReplicas int
	WorkerReplicas       int
	APIServerLoadBalancer bool
	IncludeCCMAddon      bool
}

type workloadReadinessOptions struct {
	Namespace      string
	ClusterName    string
	TestID         string
	WantNodes      int
	WantMachines   int
	InstallCNI     bool
	WaitForCCM     bool
}
```

Implementation notes:

- Default `ControlPlaneReplicas` to `1`, `WorkerReplicas` to `1`,
  `APIServerLoadBalancer` to `true`, and `IncludeCCMAddon` to `true`.
- Keep `ServiceAccountJSON` required when `IncludeCCMAddon=true`.
- Keep the Kubernetes package-install fallback behavior exactly as today. This
  milestone is a refactor, not the image-build milestone.
- Use plain Go string assembly or small `fmt.Sprintf` helpers. Do not introduce
  a YAML templating dependency for e2e tests.

### Fixture Rendering Breakdown

Create helpers with one responsibility each:

- `renderKubeadmWorkloadClusterFixture(opts kubeadmWorkloadFixtureOptions) string`
  - top-level orchestrator replacing `renderStackitKubeadmClusterFixture`
  - joins object fragments with `---`
  - validates required fields with `Expect`
- `renderCAPICluster(opts) string`
  - renders the base CAPI `Cluster`
  - includes `cluster-api-provider-stackit/cloud-provider-stackit: "true"`
    only when `IncludeCCMAddon=true`
- `renderStackitCluster(opts) string`
  - renders existing network, credentials, region, labels, and API server load
    balancer choice
- `renderKubeadmControlPlane(opts) string`
  - renders replicas, version, kubeadm init/join config, external cloud
    provider kubelet/controller-manager args, and control-plane runtime package
    install commands
- `renderStackitMachineTemplate(name string, opts, role string) string`
  - renders machine type, image, AZ, root volume, optional SSH key, optional
    security groups, network ID, and test labels
  - role can be `control-plane` or `worker` only for naming and diagnostics
- `renderWorkerMachineDeployment(opts) string`
  - renders replicas, selector, version, kubeadm bootstrap configRef, and
    infrastructureRef
- `renderWorkerKubeadmConfigTemplate(opts) string`
  - renders worker runtime package install commands and kubeadm join config
- `renderCloudProviderStackitClusterResourceSet(opts) string`
  - renders the existing CRS and Secret using `renderCloudProviderStackitAddon`
  - returns an empty string when `IncludeCCMAddon=false`

Keep the old exported-looking function name as a compatibility wrapper while
migrating:

```go
func renderStackitKubeadmClusterFixture(clusterName, testID string, cfg stackitVMConfig, kubernetesVersion string, serviceAccountJSON []byte) string {
	return renderKubeadmWorkloadClusterFixture(kubeadmWorkloadFixtureOptions{
		ClusterName: clusterName,
		TestID: testID,
		Config: cfg,
		KubernetesVersion: kubernetesVersion,
		ServiceAccountJSON: serviceAccountJSON,
		ControlPlaneReplicas: 1,
		WorkerReplicas: 1,
		APIServerLoadBalancer: true,
		IncludeCCMAddon: true,
	})
}
```

### Readiness Helper Breakdown

Create a single orchestration helper for the NodeRef success path:

```go
func waitForKubeadmWorkloadClusterReady(
	g Gomega,
	opts workloadReadinessOptions,
	workloadKubeconfig *string,
) {
	// wait StackitCluster ready
	// wait control-plane StackitMachine ready
	// extract kubeconfig
	// install CNI if requested
	// wait CCM if requested
	// wait expected StackitMachines
	// wait workload Nodes Ready
	// assert providerID alignment
}
```

Implementation notes:

- Pass `workloadKubeconfig` by pointer so callers can reuse the path for later
  scale and upgrade assertions and remove it in `defer`.
- Do not hide cleanup in this helper. Cleanup must remain explicit in each
  scenario because each test owns different cloud resources and observations.
- Keep timeouts at least as generous as the current NodeRef test:
  - 15 minutes for `StackitCluster` readiness
  - 45 minutes for initial VM readiness
  - 20 minutes for kubeconfig/API reachability
  - 25 minutes for workload Node readiness
  - 15 minutes for providerID alignment
- Keep `installWorkloadCNI` idempotence assumptions local to e2e. The helper
  should run it once during initial cluster readiness, not during every scale or
  upgrade assertion.

Add focused lower-level helpers for later milestones:

- `expectMachineDeploymentReplicas(g, namespace, name string, replicas int)`
- `expectMachineDeploymentReadyReplicas(g, namespace, name string, replicas int)`
- `workerMachinesForDeployment(g, namespace, clusterName, deploymentName string) []capiMachineItem`
- `nodeNamesForMachines(machines []capiMachineItem) []string`
- `expectWorkloadNodesGone(g, kubeconfig string, nodeNames []string)`
- `expectWorkloadNodesReadyForMachines(g, kubeconfig string, machines []capiMachineItem)`

Only implement the helpers required by the NodeRef refactor in Milestone 1.
Leave helper additions that are only used by scale or upgrade to those
milestones unless they are trivial and directly tested by the NodeRef scenario.

### Infra-Only Fixture Naming

Rename `renderStackitMachineDeploymentScaleFixture` to make its scope explicit:

```go
renderStackitInfraOnlyMachineDeploymentFixture
```

Keep a short compatibility wrapper if the rename makes the diff noisy:

```go
func renderStackitMachineDeploymentScaleFixture(...) string {
	return renderStackitInfraOnlyMachineDeploymentFixture(...)
}
```

Update `By(...)` descriptions in the current scale and upgrade tests to include
`infra-only` or `infrastructure lifecycle` so test output cannot be confused
with full workload coverage.

### Migration Order

1. Add option structs and small rendering helpers.
2. Reimplement `renderStackitKubeadmClusterFixture` as a wrapper around the new
   top-level real workload fixture renderer.
3. Add `waitForKubeadmWorkloadClusterReady` and migrate the NodeRef test to it.
4. Rename or wrap the infra-only MachineDeployment fixture.
5. Run the fast test suite.
6. Run the NodeRef billable e2e scenario once to prove the refactor preserved
   behavior. This validation is required before marking the milestone complete.

### Acceptance Criteria

- The NodeRef test body is shorter and delegates repeated setup/readiness logic
  to helpers.
- The rendered YAML for the existing NodeRef fixture is behaviorally equivalent:
  same object kinds, same labels, same cloud-provider addon, same kubeadm
  external cloud-provider settings, same runtime package-install fallback.
- Existing infra-only scale and upgrade scenarios remain independently gated by
  `STACKIT_E2E_SCALE_WORKERS=true` and `STACKIT_E2E_UPGRADE_WORKERS=true`.
- Names or log messages make clear that the current scale and upgrade scenarios
  are infrastructure lifecycle coverage, not workload Node readiness coverage.
- No API types, templates, CRDs, RBAC, or production manifests change.

### Verification

```sh
make test
```

Required billable verification:

```sh
env STACKIT_E2E_NODE_REF=true \
  KUBERNETES_VERSION=v1.35.3 \
  STACKIT_E2E_CNI=cilium \
  go test -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='align StackitMachine'
```

## Milestone 2: Full Worker Scale E2E

Goal: prove worker scale changes create and remove Ready workload Nodes, not
only STACKIT VMs.

Implementation tasks:

- Add a new independently gated test, for example
  `STACKIT_E2E_SCALE_WORKLOAD=true`.
- Render a real kubeadm workload cluster with one control-plane and one worker
  using the shared fixture helpers from Milestone 1.
- Run the common readiness sequence and assert two Ready Nodes initially.
- Scale the worker `MachineDeployment` from one to three replicas.
- Wait for:
  - `MachineDeployment.status.replicas=3`
  - `MachineDeployment.status.readyReplicas=3`
  - three worker CAPI Machines with provider IDs
  - four Ready workload Nodes total
  - all worker Machines have non-empty `status.nodeRef.name`
  - all referenced workload Nodes have matching `spec.providerID`
- Scale the worker `MachineDeployment` back to one replica.
- Wait for:
  - `MachineDeployment.status.replicas=1`
  - `MachineDeployment.status.readyReplicas=1`
  - two Ready workload Nodes total
  - removed worker Nodes disappear from the workload cluster
  - removed STACKIT VMs disappear from STACKIT
- Keep leak checks for servers and API server load balancers.
- Keep the old `STACKIT_E2E_SCALE_WORKERS=true` infra-only test if it still has
  value, but document the difference in `docs/src/testing/scale.md`.

Acceptance criteria:

- The new scale scenario can be run without running other real-cloud scenarios.
- Scale-up proves new workload Nodes become Ready.
- Scale-down proves removed worker Nodes and VMs are gone.
- Cleanup succeeds when the test fails after cluster creation.
- The billable real-cloud validation below passes before the milestone is
  marked complete.

Required billable validation:

```sh
env STACKIT_E2E_SCALE_WORKLOAD=true \
  KUBERNETES_VERSION=v1.35.3 \
  STACKIT_E2E_CNI=cilium \
  go test -timeout=90m -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='scale.*workload'
```

## Milestone 3: Full MachineDeployment Upgrade E2E

Goal: prove a worker Kubernetes version change rolls out automatically based on
real workload Node readiness.

Implementation tasks:

- Add a new independently gated test, for example
  `STACKIT_E2E_UPGRADE_WORKLOAD_WORKERS=true`.
- Render a real kubeadm workload cluster with one control-plane and at least
  two workers. Two workers make rolling behavior observable without dropping all
  worker capacity.
- Wait for the initial workload cluster to become Ready and record:
  - worker CAPI Machine names
  - worker workload Node names
  - worker provider IDs
  - worker STACKIT instance IDs
- Patch `MachineDeployment.spec.template.spec.version` from
  `STACKIT_E2E_UPGRADE_FROM` to `STACKIT_E2E_UPGRADE_TO`.
- Do not manually delete the old worker Machine as the success path. Let the
  MachineDeployment controller complete the rollout.
- Wait for:
  - `MachineDeployment.status.updatedReplicas` equals desired replicas
  - `MachineDeployment.status.readyReplicas` equals desired replicas
  - `MachineDeployment.status.unavailableReplicas` is empty or zero
  - all current worker Machines use the target Kubernetes version
  - all current worker Machines have `status.nodeRef.name`
  - workload worker Nodes are Ready and match provider IDs
  - old worker Nodes are removed from the workload cluster
  - old STACKIT VMs are deleted
- Keep the old infra-only upgrade test if useful, but rename or document it as
  infrastructure replacement and cleanup coverage.

Acceptance criteria:

- The test verifies automatic rollout completion without manual deletion of the
  old worker Machine.
- The workload API remains reachable after the rollout by using the extracted
  workload kubeconfig for all Node assertions.
- The scenario can be executed independently from scale and control-plane
  upgrade tests.
- The billable real-cloud validation below passes before the milestone is
  marked complete.

Required billable validation:

```sh
env STACKIT_E2E_UPGRADE_WORKLOAD_WORKERS=true \
  STACKIT_E2E_UPGRADE_FROM=v1.35.3 \
  STACKIT_E2E_UPGRADE_TO=v1.35.4 \
  STACKIT_E2E_CNI=cilium \
  go test -timeout=90m -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='workload.*worker.*upgrade'
```

## Milestone 4: KubeadmControlPlane Upgrade E2E

Goal: prove control-plane upgrades replace or update control-plane Machines
while the workload API remains reachable.

Implementation tasks:

- Add a new independently gated test, for example
  `STACKIT_E2E_UPGRADE_WORKLOAD_CONTROL_PLANE=true`.
- Start with a one control-plane cluster for the first implementation. Add a
  three control-plane variant later if STACKIT cost and runtime are acceptable.
- Render a real kubeadm workload cluster and wait for all initial Nodes Ready.
- Patch `KubeadmControlPlane.spec.version` from
  `STACKIT_E2E_UPGRADE_FROM` to `STACKIT_E2E_UPGRADE_TO`.
- During the upgrade, periodically assert workload API reachability through the
  kubeconfig and API server load balancer.
- Wait for:
  - `KubeadmControlPlane.status.readyReplicas` equals desired replicas
  - `KubeadmControlPlane.status.updatedReplicas` equals desired replicas if the
    field is present in the installed CAPI version
  - replacement or updated control-plane Machine reports the target version
  - control-plane workload Node is Ready
  - Machine/Node provider IDs remain aligned
  - old control-plane VM is deleted if replacement occurred
- Keep cloud leak checks for servers and load balancers.
- Do not require the old single-node control-plane workload Node object to be
  removed yet. Billable validation showed CAPI can delete the old Machine and
  STACKIT VM while the old workload Node remains `NotReady,SchedulingDisabled`;
  track that as a follow-up gap instead of hiding it in this milestone.

Acceptance criteria:

- The test fails if the workload API becomes permanently unreachable.
- The test proves the control-plane Node is Ready after upgrade.
- The scenario can be run without the worker upgrade test.
- The billable real-cloud validation below passes before the milestone is
  marked complete.

Required billable validation:

```sh
env STACKIT_E2E_UPGRADE_WORKLOAD_CONTROL_PLANE=true \
  STACKIT_E2E_UPGRADE_FROM=v1.35.3 \
  STACKIT_E2E_UPGRADE_TO=v1.35.4 \
  STACKIT_E2E_CNI=cilium \
  go test -timeout=90m -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='workload.*control.*upgrade'
```

## Milestone 5: Full ClusterClass Topology E2E

Goal: prove topology clusters can create, reach Ready Nodes, and delete without
cloud leaks.

Implementation tasks:

- Add a new independently gated test, for example
  `STACKIT_E2E_TOPOLOGY_WORKLOAD=true`.
- Ensure the management cluster used by the test has `ClusterTopology=true` for
  CAPI core and kubeadm-control-plane.
- Apply `templates/clusterclass.yaml` as part of the test setup or document it
  as a required precondition checked by the test.
- Render `templates/cluster-template-topology.yaml` through `clusterctl` or an
  equivalent local renderer using real STACKIT variables.
- Add topology-compatible `cloud-provider-stackit` addon wiring. The current
  topology template does not include the addon `ClusterResourceSet` or workload
  Secret that the non-topology template and NodeRef fixture use.
- Create a topology cluster with one control-plane and one worker.
- Install the selected CNI, wait for `cloud-provider-stackit`, wait for two
  Ready workload Nodes, and assert providerID alignment.
- Delete the topology cluster and assert:
  - CAPI resources are gone
  - `StackitMachine` resources are gone
  - STACKIT VMs are gone
  - STACKIT API server load balancer is gone
  - no tagged resources remain

Acceptance criteria:

- ClusterClass/topology coverage is no longer apply-only.
- The topology scenario proves real workload Node readiness.
- The test can be executed independently from non-topology create/delete,
  scale, and upgrade tests.

Required billable validation:

```sh
env STACKIT_E2E_TOPOLOGY_WORKLOAD=true \
  KUBERNETES_VERSION=v1.35.3 \
  STACKIT_E2E_CNI=cilium \
  go test -timeout=90m -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='topology.*workload' --ginkgo.timeout=90m
```

## Milestone 6: Documentation and CI Wiring

Goal: make the new full workload coverage discoverable and runnable without
guessing environment variables.

Implementation tasks:

- Update `docs/src/testing/scale.md` to distinguish infra-only scale from full
  workload scale.
- Update `docs/src/testing/upgrade.md` to distinguish infra-only worker
  replacement from full worker and control-plane upgrade scenarios.
- Update `docs/src/usage/clusterclass.md` after topology addon wiring lands.
- Update `todo.md` and `docs/src/development/roadmap.md` as each scenario is
  implemented.
- Add make targets or documented command snippets for each independent scenario.
- Keep billable real-cloud scenarios opt-in. CI should run them only in a
  dedicated STACKIT project with credentials, cleanup permissions, and cost
  controls.

Acceptance criteria:

- A developer can run any scenario independently with one documented command.
- Docs do not imply infra-only coverage proves workload Node readiness.
- The roadmap reflects completed and remaining e2e scope accurately.

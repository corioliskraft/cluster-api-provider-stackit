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
- `todo.md` still tracks the missing scale Node readiness/removal,
  automatic MachineDeployment rollout completion, KubeadmControlPlane upgrade,
  workload API reachability during upgrade, and full ClusterClass
  create/ready/delete coverage.
- `templates/clusterclass.yaml` and `templates/cluster-template-topology.yaml`
  exist, but there is no real-cloud topology e2e scenario in `test/e2e`.
  `docs/src/usage/clusterclass.md` also states that topology wiring currently
  lacks the `cloud-provider-stackit` addon wiring needed for Ready Nodes.

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

Implementation tasks:

- Refactor `renderStackitKubeadmClusterFixture` into smaller helpers that can
  render:
  - base `Cluster`
  - `StackitCluster`
  - `KubeadmControlPlane`
  - control-plane `StackitMachineTemplate`
  - worker `MachineDeployment`
  - worker `StackitMachineTemplate`
  - worker `KubeadmConfigTemplate`
  - `cloud-provider-stackit` `ClusterResourceSet` and Secret
- Add options for initial worker replica count, Kubernetes version, CNI choice,
  API server load balancer, and test labels.
- Keep the current static bootstrap fixture under a clearly named helper such
  as `renderStackitInfraOnlyMachineDeploymentFixture` so infra-only tests remain
  available and intentionally scoped.
- Add a helper that performs the common readiness sequence:
  - wait for `StackitCluster.status.ready=true`
  - wait for control-plane VM readiness
  - extract workload kubeconfig
  - install workload CNI
  - wait for `cloud-provider-stackit`
  - wait for expected workload Nodes Ready
  - assert Machine/Node providerID alignment
- Add helper functions for counting worker Nodes by MachineDeployment labels or
  by CAPI Machine `status.nodeRef`.

Acceptance criteria:

- Existing `STACKIT_E2E_NODE_REF=true` scenario still passes with the refactored
  helpers.
- Existing infra-only scale and upgrade tests either still compile unchanged or
  are explicitly renamed to show they are infrastructure lifecycle tests.
- `make test` passes.

Suggested command:

```sh
make test
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

Suggested command:

```sh
env STACKIT_E2E_SCALE_WORKLOAD=true \
  KUBERNETES_VERSION=v1.35.3 \
  STACKIT_E2E_CNI=cilium \
  go test -tags=e2e ./test/e2e -v -ginkgo.v \
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

Suggested command:

```sh
env STACKIT_E2E_UPGRADE_WORKLOAD_WORKERS=true \
  STACKIT_E2E_UPGRADE_FROM=v1.35.3 \
  STACKIT_E2E_UPGRADE_TO=v1.35.4 \
  STACKIT_E2E_CNI=cilium \
  go test -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='upgrade.*worker.*workload'
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

Acceptance criteria:

- The test fails if the workload API becomes permanently unreachable.
- The test proves the control-plane Node is Ready after upgrade.
- The scenario can be run without the worker upgrade test.

Suggested command:

```sh
env STACKIT_E2E_UPGRADE_WORKLOAD_CONTROL_PLANE=true \
  STACKIT_E2E_UPGRADE_FROM=v1.35.3 \
  STACKIT_E2E_UPGRADE_TO=v1.35.4 \
  STACKIT_E2E_CNI=cilium \
  go test -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='upgrade.*control.*workload'
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

Suggested command:

```sh
env STACKIT_E2E_TOPOLOGY_WORKLOAD=true \
  KUBERNETES_VERSION=v1.35.3 \
  STACKIT_E2E_CNI=cilium \
  go test -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='topology.*workload'
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

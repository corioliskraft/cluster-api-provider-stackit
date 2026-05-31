# Upgrade

Upgrade coverage has three independently gated e2e paths.

The infra-only path, gated by `STACKIT_E2E_UPGRADE_WORKERS=true`, focuses on
MachineDeployment worker replacement with a static bootstrap Secret.

Validated infra-only behavior:

- changing the MachineDeployment Kubernetes version creates replacement workers
- replacement worker VMs become ready at the infrastructure level
- old worker VMs are deleted when the old Machines are removed

The full workload worker path, gated by
`STACKIT_E2E_UPGRADE_WORKLOAD_WORKERS=true`, uses the real kubeadm workload
fixture with CNI and `cloud-provider-stackit`.

Validated workload worker behavior:

- initial control-plane and worker Nodes become Ready
- changing the MachineDeployment Kubernetes version completes a CAPI rollout
- replacement worker Nodes become Ready and match Machine provider IDs
- old worker Nodes are removed from the workload cluster
- old worker VMs are deleted from STACKIT
- no tagged cloud resources remain orphaned

The full workload control-plane path, gated by
`STACKIT_E2E_UPGRADE_WORKLOAD_CONTROL_PLANE=true`, uses the real kubeadm
workload fixture with CNI, `cloud-provider-stackit`, and an API server load
balancer.

Validated workload control-plane behavior:

- initial control-plane and worker Nodes become Ready
- changing the KubeadmControlPlane Kubernetes version completes a CAPI rollout
- workload API reachability recovers through the API server load balancer
- upgraded control-plane Node is Ready and matches Machine provider IDs
- old control-plane VM is deleted from STACKIT if replacement occurred
- no tagged cloud resources remain orphaned

Pending behavior:

- old single-node control-plane Node object cleanup after KubeadmControlPlane
  replacement. The billable Milestone 4 validation observed the old Machine and
  VM being deleted while the old workload Node remained
  `NotReady,SchedulingDisabled`.
- highly available KubeadmControlPlane upgrade e2e with three control-plane
  replicas
- continuous workload API reachability sampling outside the rollout wait loop

See [Cluster Upgrade](../usage/upgrade.md) for the user-facing upgrade flow.

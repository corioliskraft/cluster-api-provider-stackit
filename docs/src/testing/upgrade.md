# Upgrade

Upgrade coverage has two independently gated e2e paths.

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

Pending behavior:

- KubeadmControlPlane upgrade e2e
- workload-cluster reachability throughout the upgrade

See [Cluster Upgrade](../usage/upgrade.md) for the user-facing upgrade flow.

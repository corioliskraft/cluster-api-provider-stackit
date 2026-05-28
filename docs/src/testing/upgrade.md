# Upgrade

Upgrade coverage currently focuses on MachineDeployment worker replacement.

Validated behavior:

- changing the MachineDeployment Kubernetes version creates replacement workers
- replacement worker VMs become ready at the infrastructure level
- old worker VMs are deleted when the old Machines are removed

Pending behavior:

- automatic MachineDeployment rolling completion with real Node readiness
- KubeadmControlPlane upgrade e2e
- workload-cluster reachability throughout the upgrade

See [Cluster Upgrade](../usage/upgrade.md) for the user-facing upgrade flow.

# Cluster API Provider STACKIT

`cluster-api-provider-stackit` is a Cluster API infrastructure provider for
STACKIT Cloud. It creates and deletes STACKIT infrastructure for workload
clusters while using the upstream Cluster API core, kubeadm bootstrap provider,
and kubeadm control-plane provider.

The provider is intentionally scoped as an infrastructure provider. It manages:

- `StackitCluster` and `StackitClusterTemplate`
- `StackitMachine` and `StackitMachineTemplate`
- Existing STACKIT networks
- STACKIT servers
- Optional provider-managed API server load balancers

The implementation is moving from MVP toward a contract-correct, reproducibly
testable, clusterctl-usable provider. This book documents the current behavior,
the validated flows, and the remaining gaps.

## Current Status

Working areas:

- API types and controllers are scaffolded and tested.
- VM create/delete and API server load balancer flows are implemented.
- Provider IDs match `cloud-provider-stackit`: `stackit://<server-id>`.
- `clusterctl` local release packaging is available.
- Real STACKIT VM e2e scenarios exist behind opt-in flags.
- Worker scale, upgrade replacement, failure domains, and ClusterClass templates
  have focused coverage or validation.

Known gaps:

- Workload-cluster Node readiness depends on finishing the
  `cloud-provider-stackit` addon flow.
- Full ClusterClass create/ready/delete e2e is still pending.
- Release distribution is not finalized between installer YAML, Helm, or both.

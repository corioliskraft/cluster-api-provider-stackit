# Overview

The provider creates STACKIT infrastructure for Cluster API workload clusters.
The management cluster usually runs locally with kind, while workload-cluster
machines run as STACKIT Compute Engine servers.

Provider stack:

| Role | Provider |
| --- | --- |
| Core | `cluster-api` |
| Bootstrap | kubeadm / CABPK |
| Control plane | kubeadm / KCP |
| Infrastructure | `cluster-api-provider-stackit` |

The provider does not implement its own bootstrap or control-plane logic. It
receives bootstrap data from Cluster API bootstrap Secrets and passes it to
STACKIT servers as cloud-init user data.

## MVP Flow

The target flow is:

```sh
kind create cluster --name capi-stackit
clusterctl init --bootstrap kubeadm --control-plane kubeadm
make install
make run
kubectl apply -f rendered-cluster.yaml
```

The expected result is a Cluster API `Cluster`, `Machine` resources,
`StackitCluster`, and `StackitMachine` resources that become ready, with all
provider-created STACKIT resources deleted when the Cluster is deleted.

## Production Readiness Gaps

The provider is a solid MVP, but it is not production-ready yet. It can use
existing STACKIT networks, create and delete VMs, set provider IDs, and manage
an optional API server load balancer. The remaining gaps are mostly around
workload-cluster readiness, cloud-provider integration, release/distribution,
and Cluster API contract details.

- Workload-cluster readiness is still open. The full workload-cluster e2e path
  and Node readiness with `cloud-provider-stackit` are still pending.
- The cloud controller manager is not part of the default cluster template.
  `templates/cluster-template.yaml` sets `cloud-provider=external`, but it does
  not install `cloud-provider-stackit`. Without the CCM, Nodes can remain
  uninitialized and `Node.spec.providerID` / CAPI `NodeRef` alignment can be
  missing. A separate addon manifest exists in
  `templates/addons/cloud-provider-stackit.yaml`.
- The release story is not finalized. Local `clusterctl` assets can be
  generated, but the final distribution path is still open. Production use
  needs versioned images, published `clusterctl` assets, upgrade guidance,
  support matrix, and preferably image signing / SBOMs.
- Existing cloud resources are currently required. The provider does not create
  STACKIT networks, security groups, SSH keys, or images. Production users need
  a clear prerequisite flow, for example Terraform modules or explicit
  documented setup steps, unless the provider grows first-class management for
  those resources.
- Failure domains are currently derived from the configured region as
  `<region>-1`, `<region>-2`, and `<region>-3`. Production behavior should
  discover available STACKIT zones dynamically and publish only real failure
  domains.
- Admission webhooks are missing. CRD validation exists, but there are no
  defaulting / validation webhooks for immutability, safe updates, template
  fields, cross-resource checks, or clearer user-facing validation failures.
- Operational hardening is still needed. The manager deployment is still close
  to the Kubebuilder scaffold in places, and production use should define
  resource sizing, metrics TLS, HA/replica behavior, alerting, and runbooks.

## Full Cluster API Support

For full Cluster API support, the provider should complete the following areas:

- Add CAPI core RBAC aggregation. The CRDs already carry the
  `cluster.x-k8s.io/v1beta2: v1alpha1` contract label, but the provider should
  also ship `cluster.x-k8s.io/aggregate-to-manager: "true"` RBAC for
  `StackitCluster`, `StackitMachine`, and their templates so CAPI core
  controllers can manage owner references, labels, and ClusterClass-generated
  infrastructure resources correctly.
- Respect `Machine.spec.failureDomain`. The provider currently uses
  `StackitMachine.spec.availabilityZone`. For full CAPI behavior, if CAPI sets
  `Machine.spec.failureDomain`, the infrastructure machine must be placed in
  that failure domain. The provider should also consider surfacing the actual
  placement through `StackitMachine.status.failureDomain`.
- Complete ClusterClass support. `StackitClusterTemplate` and
  `StackitMachineTemplate` exist, but the template resources should support
  template metadata for labels and annotations. Full create/ready/delete e2e
  coverage for topology clusters is still required.
- Provide a tested addon flow for `cloud-provider-stackit` and the CNI. Either
  the provider should ship a `clusterctl`-friendly addon template, or the docs
  should define a tested post-create installation path that reliably clears the
  external cloud-provider node taint and aligns provider IDs.
- Harden scale, upgrade, remediation, and deletion behavior. Worker scale and
  replacement upgrade coverage exists, but full support should include control
  plane replacement, MachineHealthCheck remediation, interrupted deletes,
  orphan cleanup, and cloud-side drift handling.

The relevant Cluster API contract areas are the InfraCluster and InfraMachine
contracts. `StackitCluster` must provide the control plane endpoint,
initialization state, failure domains, and conditions. `StackitMachine` must
provide the provider ID, initialization state, and should surface addresses and
failure-domain placement where available.

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

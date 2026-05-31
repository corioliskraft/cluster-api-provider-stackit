# Management Cluster

Create an isolated kind management cluster:

```sh
kind create cluster --name capi-stackit
kubectl config use-context kind-capi-stackit
```

Install Cluster API core, kubeadm bootstrap, and kubeadm control-plane providers.
Use the local clusterctl config so the `ClusterResourceSet` feature is enabled;
the default non-topology cluster template uses it to install
`cloud-provider-stackit` into workload clusters:

```sh
clusterctl init \
  --config hack/clusterctl-local.yaml \
  --core cluster-api \
  --bootstrap kubeadm \
  --control-plane kubeadm
```

Build and deploy this infrastructure provider:

```sh
export IMG=cluster-api-provider-stackit:dev
make docker-build IMG="${IMG}"
kind load docker-image "${IMG}" --name capi-stackit
make deploy IMG="${IMG}"
```

Wait for the controller manager:

```sh
kubectl rollout status \
  -n cluster-api-provider-stackit-system \
  deployment/cluster-api-provider-stackit-controller-manager
```

For ClusterClass and topology clusters, CAPI must also run with
`ClusterTopology=true`. The same local `hack/clusterctl-local.yaml` enables this
feature gate for clusterctl-based local installs.

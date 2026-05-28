# Management Cluster

Create an isolated kind management cluster:

```sh
kind create cluster --name capi-stackit
kubectl config use-context kind-capi-stackit
```

Install Cluster API core, kubeadm bootstrap, and kubeadm control-plane providers:

```sh
clusterctl init \
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

For ClusterClass and topology clusters, initialize or patch CAPI with
`ClusterTopology=true`. The local `hack/clusterctl-local.yaml` enables this
feature gate for clusterctl-based local installs.

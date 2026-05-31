# ClusterClass

ClusterClass support is provided by:

- `templates/clusterclass.yaml`
- `templates/cluster-template-topology.yaml`

The ClusterClass wires topology variables into STACKIT templates:

- Kubernetes version
- Control-plane replica count
- Worker replica count
- Machine type
- Image ID
- Region
- Network ID
- Credentials Secret name
- Additional STACKIT cloud labels
- Development fallback `preKubeadmCommands`

Apply the ClusterClass:

```sh
kubectl apply -f templates/clusterclass.yaml
```

Render and apply a topology Cluster:

```sh
export KUBERNETES_VERSION=v1.35.3
export KUBERNETES_APT_REPOSITORY_MINOR=v1.35
export STACKIT_SERVICE_ACCOUNT_JSON_B64="$(base64 < serviceaccount.json | tr -d '\n')"
export STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE=ghcr.io/stackitcloud/cloud-provider-stackit/cloud-controller-manager:v1.35.3

clusterctl generate cluster "${CLUSTER_NAME}" \
  --from templates/cluster-template-topology.yaml \
  --kubernetes-version "${KUBERNETES_VERSION}" \
  --control-plane-machine-count 1 \
  --worker-machine-count 1 \
  > "${CLUSTER_NAME}-topology.yaml"

kubectl apply -f "${CLUSTER_NAME}-topology.yaml"
```

The management cluster must run CAPI core and kubeadm-control-plane with
`ClusterTopology=true`. Otherwise the admission webhooks reject `ClusterClass`,
`KubeadmControlPlaneTemplate`, and `Cluster.spec.topology`.

The topology cluster template includes a `cloud-provider-stackit`
`ClusterResourceSet` addon and a workload-cluster Secret for the cloud
controller manager. It also passes the same development fallback
`preKubeadmCommands` used by the real e2e workload fixture so generic Ubuntu
images can install `containerd`, `kubelet`, `kubeadm`, and `kubectl` before
kubeadm runs. For production, prefer kubeadm-ready images and manage addons
through your normal Helm, GitOps, or addon-provider workflow.

Run the topology workload e2e path independently only when billable STACKIT
validation is intended:

```sh
env STACKIT_E2E_TOPOLOGY_WORKLOAD=true \
  KUBERNETES_VERSION=v1.35.3 \
  STACKIT_E2E_CNI=cilium \
  go test -timeout=90m -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='topology.*workload' --ginkgo.timeout=90m
```

Equivalent make target:

```sh
make test-e2e-workload-topology
```

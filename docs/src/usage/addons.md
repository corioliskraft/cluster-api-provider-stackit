# Workload Addons

This provider supports a tested addon flow for unmanaged STACKIT workload
clusters:

- `cloud-provider-stackit` is rendered into the workload cluster by the cluster
  template through a Cluster API `ClusterResourceSet`.
- The CNI is installed after the workload API server is reachable.
- The default tested CNI path uses Cilium with the checked-in values in
  `templates/addons/cilium-values.yaml`.

This split is intentional. The cloud provider must run early so Kubernetes
Nodes get STACKIT provider IDs and the external cloud-provider taint is cleared.
The CNI is a cluster operator choice because routing, network policy, MTU,
IPAM, and upgrade strategy are environment-specific.

## Cloud Provider

The classic and topology cluster templates include a
`stackit-workload-cloud-provider-stackit` `ClusterResourceSet` and Secret. The
resource set applies `templates/addons/cloud-provider-stackit.yaml` into the
workload cluster.

The addon template installs:

- the `stackit-cloud-controller-manager` ServiceAccount, RBAC, ConfigMap,
  Secret, and Deployment in `kube-system`
- the `--cloud-provider=stackit` controller configuration
- the `cloud-node-controller`, `cloud-node-lifecycle-controller`, and
  `service-lb-controller`
- tolerations required to run before Nodes are initialized

The kubeadm templates configure the control plane and kubelets for an external
cloud provider. When `cloud-provider-stackit` is healthy, it sets STACKIT
provider IDs on Nodes and removes the
`node.cloudprovider.kubernetes.io/uninitialized` taint.

Before creating the cluster, ensure the management cluster has
`ClusterResourceSet` enabled. The local clusterctl configuration in
`hack/clusterctl-local.yaml` enables it with `CLUSTER_RESOURCE_SET=true`.

Set the cloud provider inputs before rendering the template:

```sh
export STACKIT_SERVICE_ACCOUNT_JSON_FILE=./sa/cluster-api-provider-stackit-serviceaccount.json
export STACKIT_SERVICE_ACCOUNT_JSON_B64="$(base64 < "${STACKIT_SERVICE_ACCOUNT_JSON_FILE}" | tr -d '\n')"
export STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE="ghcr.io/stackitcloud/cloud-provider-stackit:<version>"
```

The cloud-provider image minor version must match the Kubernetes minor version
of the workload cluster. Use the version helper before rendering:

```sh
hack/validate-stackit-versions.sh
```

## CNI

After the workload API server is reachable, retrieve the kubeconfig:

```sh
clusterctl get kubeconfig "${CLUSTER_NAME}" \
  --namespace "${NAMESPACE}" \
  > "${CLUSTER_NAME}.kubeconfig"
```

Install the default tested Cilium CNI:

```sh
make install-workload-cni \
  WORKLOAD_KUBECONFIG="${CLUSTER_NAME}.kubeconfig"
```

The helper waits for the Cilium operator, Cilium DaemonSet, and Cilium Envoy
DaemonSet when present.

To install Calico instead:

```sh
make install-workload-cni \
  WORKLOAD_KUBECONFIG="${CLUSTER_NAME}.kubeconfig" \
  STACKIT_WORKLOAD_CNI=calico
```

To apply a custom CNI manifest:

```sh
make install-workload-cni \
  WORKLOAD_KUBECONFIG="${CLUSTER_NAME}.kubeconfig" \
  CNI_MANIFEST=./my-cni.yaml
```

Custom manifests are applied as-is. The helper cannot know which workload
resources prove that an arbitrary CNI is healthy, so custom CNI users must wait
for their own rollout resources.

## Verification

Verify that the cloud provider Deployment rolled out:

```sh
kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" \
  -n kube-system rollout status deployment/stackit-cloud-controller-manager \
  --timeout=5m
```

Verify Cilium rollout for the default path:

```sh
kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" \
  -n kube-system rollout status deployment/cilium-operator --timeout=10m
kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" \
  -n kube-system rollout status daemonset/cilium --timeout=10m
kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" \
  -n kube-system rollout status daemonset/cilium-envoy --timeout=10m
```

Verify Nodes are Ready and have STACKIT provider IDs:

```sh
kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" get nodes
kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" \
  get nodes \
  -o custom-columns=NAME:.metadata.name,PROVIDER_ID:.spec.providerID
```

Verify the external cloud-provider taint is gone:

```sh
kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" \
  get nodes -o json \
  | jq -e '[.items[].spec.taints[]? | select(.key == "node.cloudprovider.kubernetes.io/uninitialized")] | length == 0'
```

Verify CAPI Machines and workload Nodes agree on STACKIT provider IDs:

```sh
kubectl -n "${NAMESPACE}" \
  get machines,stackitmachines \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}" \
  -o custom-columns=KIND:.kind,NAME:.metadata.name,PROVIDER_ID:.spec.providerID,NODE:.status.nodeRef.name
kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" \
  get nodes \
  -o custom-columns=NAME:.metadata.name,PROVIDER_ID:.spec.providerID
```

The e2e workload tests validate this flow by waiting for the embedded
`cloud-provider-stackit` rollout, installing CNI, waiting for Ready Nodes,
checking the external cloud-provider taint is removed, and verifying CAPI
Machine, `StackitMachine`, and Node provider IDs align.

## Production Addon Management

`ClusterResourceSet` is useful for static bootstrap addons such as this tested
cloud-provider installation, but it is not a full addon lifecycle manager. For
production CNI management, prefer Helm, GitOps, or a Cluster API addon provider
that handles upgrades, rollback, and drift reconciliation.

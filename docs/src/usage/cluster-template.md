# Cluster Template

`templates/cluster-template.yaml` is the default clusterctl template for a
non-topology workload cluster. It renders:

- `Cluster`
- `StackitCluster`
- `KubeadmControlPlane`
- control-plane `StackitMachineTemplate`
- `MachineDeployment`
- worker `StackitMachineTemplate`
- worker `KubeadmConfigTemplate`
- `ClusterResourceSet` and resource-set `Secret` for `cloud-provider-stackit`

Render a cluster:

```sh
clusterctl generate cluster "${CLUSTER_NAME}" \
  --from templates/cluster-template.yaml \
  --kubernetes-version "${KUBERNETES_VERSION}" \
  --control-plane-machine-count "${CONTROL_PLANE_MACHINE_COUNT}" \
  --worker-machine-count "${WORKER_MACHINE_COUNT}" \
  > "${CLUSTER_NAME}.yaml"
```

Apply the rendered manifest:

```sh
kubectl apply -f "${CLUSTER_NAME}.yaml"
```

The template configures kubeadm for an external cloud provider by setting
`cloud-provider=external` on kubelet and controller-manager.

The template also installs `cloud-provider-stackit` into the workload cluster
through Cluster API `ClusterResourceSet`. The management cluster must have the
ClusterResourceSet feature enabled before applying the generated cluster. For
local validation, `hack/clusterctl-local.yaml` sets `CLUSTER_RESOURCE_SET=true`.
The topology ClusterClass template follows the same pattern and also includes
the cloud-provider addon wiring.

Set `STACKIT_SERVICE_ACCOUNT_JSON_B64` to a single-line base64 encoding of the
STACKIT service account JSON. Set `STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE` to a
`cloud-provider-stackit` image whose minor version matches
`KUBERNETES_VERSION`; supported workload cluster minors are v1.33.x through
v1.36.x. Use `hack/validate-stackit-versions.sh` before rendering to catch
unsupported or mismatched versions.

The template installs the cloud controller manager, not a CNI. After the
workload API is reachable, install a CNI that matches the configured pod/service
CIDRs and network policy expectations before expecting Nodes to become Ready.
For a reproducible development path, use the helper documented in
[Workload CNI](cni.md).

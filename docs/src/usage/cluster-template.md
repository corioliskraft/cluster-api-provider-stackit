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
`cloud-provider=external` on kubelet, API server, and controller-manager where
needed.

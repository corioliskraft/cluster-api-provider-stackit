# Cluster Upgrade

Create the workload cluster with the source version:

```sh
export KUBERNETES_VERSION=v1.31.0
clusterctl generate cluster "${CLUSTER_NAME}" \
  --from templates/cluster-template.yaml \
  > "${CLUSTER_NAME}.yaml"
kubectl apply -f "${CLUSTER_NAME}.yaml"
```

Upgrade the control plane first:

```sh
kubectl patch kubeadmcontrolplane "${CLUSTER_NAME}-control-plane" \
  -n "${NAMESPACE}" \
  --type merge \
  -p '{"spec":{"version":"v1.32.0"}}'
```

Then upgrade the workers:

```sh
kubectl patch machinedeployment "${CLUSTER_NAME}-md-0" \
  -n "${NAMESPACE}" \
  --type merge \
  -p '{"spec":{"template":{"spec":{"version":"v1.32.0"}}}}'
```

Watch rollout state:

```sh
kubectl get kubeadmcontrolplane,machinedeployment,machineset,machine \
  -n "${NAMESPACE}" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}" \
  --watch
```

The current tested scope verifies MachineDeployment replacement behavior and VM
cleanup. Full KubeadmControlPlane and Node readiness verification depends on
completing workload-cluster bootstrap and cloud-provider integration.

See the root `cluster-upgrade.md` tutorial for the detailed step-by-step flow.

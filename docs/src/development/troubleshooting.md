# Troubleshooting

## ClusterClass is rejected

If the API server rejects `ClusterClass`, `KubeadmControlPlaneTemplate`, or
`Cluster.spec.topology`, check the CAPI manager feature gates:

```sh
kubectl -n capi-system get deployment capi-controller-manager \
  -o jsonpath='{.spec.template.spec.containers[0].args}'

kubectl -n capi-kubeadm-control-plane-system \
  get deployment capi-kubeadm-control-plane-controller-manager \
  -o jsonpath='{.spec.template.spec.containers[0].args}'
```

Both must include `ClusterTopology=true`.

## Machine waits for bootstrap data

Check the owning `Machine` and bootstrap Secret:

```sh
kubectl describe machine "${MACHINE_NAME}" -n "${NAMESPACE}"
kubectl get secret -n "${NAMESPACE}"
```

`StackitMachine` intentionally does not contain user data directly. Bootstrap
data comes from the CAPI bootstrap Secret.

STACKIT CLI might also be valueable to check server logs during boot: `stackit server log <server-id>`

## Cloud resources remain after a failed e2e run

Run direct cleanup by e2e tags:

```sh
make cleanup-stackit
```

Then verify no tagged STACKIT VMs or load balancers remain.

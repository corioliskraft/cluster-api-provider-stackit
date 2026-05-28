# Cleanup

Delete a workload cluster through Cluster API:

```sh
kubectl delete cluster "${CLUSTER_NAME}" -n "${NAMESPACE}"
```

Then check Kubernetes resources:

```sh
kubectl get machine,stackitmachine,stackitcluster \
  -n "${NAMESPACE}" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}"
```

Real e2e resources are labeled so they can be cleaned up directly through the
STACKIT API if Kubernetes cleanup fails:

- `cluster-api-provider-stackit/e2e=true`
- `cluster-api-provider-stackit/test-id=<unique-id>`
- `cluster.x-k8s.io/cluster-name=<cluster-name>`
- `cluster.x-k8s.io/cluster-namespace=<namespace>`

Run direct cloud cleanup for tagged e2e resources:

```sh
make cleanup-stackit
```

The cleanup path must not depend on Kubernetes objects. It should find and
delete resources directly by STACKIT labels.

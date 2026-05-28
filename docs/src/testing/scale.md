# Scale

Worker scale coverage exercises a `MachineDeployment` scale up and down:

```sh
kubectl scale machinedeployment "${CLUSTER_NAME}-md-0" \
  -n "${NAMESPACE}" \
  --replicas=3
```

Then back down:

```sh
kubectl scale machinedeployment "${CLUSTER_NAME}-md-0" \
  -n "${NAMESPACE}" \
  --replicas=1
```

Validated behavior:

- new `Machine` and `StackitMachine` objects are created
- provider IDs are assigned
- new STACKIT VMs are created
- scaled-down VMs are deleted
- no tagged cloud resources remain orphaned

Worker Node readiness/removal is still tied to the remaining workload-cluster
cloud-provider bootstrap work.

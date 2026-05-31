# Scale

Worker scale coverage has two independently gated e2e paths.

The infra-only path, gated by `STACKIT_E2E_SCALE_WORKERS=true`, exercises a
`MachineDeployment` scale up and down with a static bootstrap Secret:

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

Validated infra-only behavior:

- new `Machine` and `StackitMachine` objects are created
- provider IDs are assigned
- new STACKIT VMs are created
- scaled-down VMs are deleted
- no tagged cloud resources remain orphaned

The full workload path, gated by `STACKIT_E2E_SCALE_WORKLOAD=true`, uses the
real kubeadm workload fixture with CNI and `cloud-provider-stackit`.

Validated workload behavior:

- initial control-plane and worker Nodes become Ready
- scaling workers from 1 to 3 creates Ready workload Nodes
- the MachineDeployment reports the expected ready replicas
- scaling workers back to 1 removes the extra workload Nodes
- scaled-down STACKIT VMs are deleted
- no tagged cloud resources remain orphaned

Run the full workload path only when billable STACKIT e2e validation is
intended.

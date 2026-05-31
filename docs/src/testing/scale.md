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

Run the infra-only path independently:

```sh
env STACKIT_E2E_SCALE_WORKERS=true \
  KUBERNETES_VERSION=v1.35.3 \
  go test -timeout=90m -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='scale a worker MachineDeployment' --ginkgo.timeout=90m
```

Run the full workload path independently only when billable STACKIT e2e
validation is intended:

```sh
env STACKIT_E2E_SCALE_WORKLOAD=true \
  KUBERNETES_VERSION=v1.35.3 \
  STACKIT_E2E_CNI=cilium \
  go test -timeout=90m -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='scale workload worker Nodes' --ginkgo.timeout=90m
```

Equivalent make target:

```sh
make test-e2e-workload-scale
```

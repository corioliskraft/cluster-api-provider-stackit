# Cluster API Contract

The provider implements the Cluster API infrastructure contract for:

- `StackitCluster`
- `StackitClusterTemplate`
- `StackitMachine`
- `StackitMachineTemplate`

Important contract behavior:

- `StackitCluster.status.initialization.provisioned` is set when cluster
  infrastructure is ready.
- `StackitMachine.status.initialization.provisioned` is set when the VM is
  provisioned.
- `StackitMachine.spec.providerID` is set to the provider-compatible value.
- Conditions include `observedGeneration`.
- Paused clusters or paused resources must not trigger cloud API calls.
- Finalizers clean up provider-owned resources before Kubernetes object removal.

`StackitCluster.spec.controlPlaneEndpoint` is used for the Cluster API
infrastructure cluster contract when the provider manages the API server load
balancer endpoint.

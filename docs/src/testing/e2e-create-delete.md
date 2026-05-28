# Create/Delete E2E

The create/delete e2e scenario verifies that a real STACKIT cluster can be
created and deleted without leaking provider-owned resources.

Target scenario:

```text
1 control-plane Machine
1 worker Machine
provider-managed API server load balancer
delete Cluster
no Machines, StackitMachines, VMs, load balancers, or finalizer leaks
```

Run real VM e2e only with explicit opt-in:

```sh
export STACKIT_E2E_CREATE_VMS=true
go test -tags=e2e ./test/e2e -v -ginkgo.v
```

The test resources must be tagged with the e2e labels documented in
[Cleanup](../usage/cleanup.md).

# Controllers

`StackitClusterReconciler` is responsible for:

- Reading credentials
- Looking up the configured network
- Managing the optional API server load balancer
- Publishing failure domains
- Updating readiness and contract status
- Cleaning up provider-managed load balancers on deletion

`StackitMachineReconciler` is responsible for:

- Waiting for bootstrap data
- Creating STACKIT servers
- Setting provider IDs and addresses
- Registering control-plane machines as API server load balancer targets
- Deleting servers and load balancer targets on teardown

Reconciliation must be idempotent. Re-running the same reconcile loop should be
safe and should not create duplicate cloud resources.

# Failure Domains

The provider publishes STACKIT availability zones as Cluster API failure
domains on `StackitCluster.status.failureDomains`.

For a region such as `eu01`, the current model exposes:

```text
eu01-1
eu01-2
eu01-3
```

`StackitMachine.spec.availabilityZone` is validated against the published
failure domains when they are available. Invalid zones result in clear
conditions instead of cloud API calls.

The existing single-zone templates continue to work. More advanced multi-AZ
scheduling remains intentionally limited for now.

---
A future improvement:

Failure domains are currently derived from the configured region as `<region>-1`, `<region>-2`, and `<region>-3`. Production behavior should discover available STACKIT zones dynamically and publish only real failure domains.

Furthermore, Respect `Machine.spec.failureDomain`. The provider currently uses `StackitMachine.spec.availabilityZone`. For full CAPI behavior, if CAPI sets `Machine.spec.failureDomain`, the infrastructure machine must be placed in that failure domain. The provider should also consider surfacing the actual placement through `StackitMachine.status.failureDomain`.

We should document FailureDomain configuration for Controlplanes and Worker nodes.

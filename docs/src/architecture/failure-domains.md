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

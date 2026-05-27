# STACKIT ProviderID Compatibility

The providerID format is verified against the local `cloud-provider-stackit`
repository at:

```text
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/cloud-provider-stackit
```

Relevant reference points:

- `pkg/ccm/instances.go`: `Instances.makeInstanceID` returns `stackit://<server-id>`.
- `pkg/ccm/instances.go`: `instanceIDFromProviderID` parses `stackit://<server-id>` with no project or region component.
- `pkg/ccm/instances.go`: `getInstance` resolves nodes with `GetServer(projectID, region, serverID)`, where project and region come from the cloud-controller-manager configuration.
- `pkg/ccm/instances_test.go`: the "new providerID" table entry expects `stackit://hello-server`.

`cluster-api-provider-stackit` therefore writes:

```text
StackitMachine.spec.providerID = stackit://<server-id>
StackitMachine.status.providerID = stackit://<server-id>
```

Cluster API then surfaces `StackitMachine.spec.providerID` to
`Machine.spec.providerID`, and `cloud-provider-stackit` uses the same value for
`Node.spec.providerID`. Project ID and region are intentionally not encoded in
the providerID.

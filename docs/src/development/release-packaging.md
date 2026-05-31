# Release Packaging

The provider currently supports local clusterctl release asset generation:

```sh
make clusterctl-release IMG=<registry>/cluster-api-provider-stackit:<tag>
```

Generated files:

- `infrastructure-components.yaml`
- `metadata.yaml`
- `clusterclass.yaml`
- `cluster-template.yaml`
- `cluster-template-development.yaml`
- `cluster-template-topology.yaml`
- `addons/*.yaml`

The release directory is:

```text
dist/clusterctl/infrastructure-stackit/v0.1.0/
```

Installer YAML can also be generated:

```sh
make build-installer IMG=<registry>/cluster-api-provider-stackit:<tag>
```

The final release path is still open: installer YAML, Helm chart, or both.

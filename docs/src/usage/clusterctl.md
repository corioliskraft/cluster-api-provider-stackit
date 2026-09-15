# clusterctl

Each release publishes `infrastructure-components.yaml`, `metadata.yaml`, and
cluster templates as GitHub Release assets.

## Install a published release

Create `clusterctl.yaml` with the published release URL. Replace `<version>`
with the release tag you want to install.

```yaml
providers:
  - name: stackit
    url: https://github.com/stackitcloud/cluster-api-provider-stackit/releases/download/v<version>/infrastructure-components.yaml
    type: InfrastructureProvider
```

Install the provider:

```sh
clusterctl init --config clusterctl.yaml --infrastructure stackit
```

## Use a local provider repository

For local development, package the provider as a local clusterctl repository:

```sh
make clusterctl-release IMG=<registry>/cluster-api-provider-stackit:<tag>
```

This writes release assets under:

```text
dist/clusterctl/infrastructure-stackit/v0.1.0/
```

Use the local clusterctl config:

```sh
export STACKIT_CLUSTERCTL_REPOSITORY="$(pwd)/dist/clusterctl"

clusterctl init \
  --config hack/clusterctl-local.yaml \
  --core cluster-api \
  --bootstrap kubeadm \
  --control-plane kubeadm \
  --infrastructure stackit:v0.1.0
```

`hack/clusterctl-local.yaml` also sets `CLUSTER_TOPOLOGY: "true"` so ClusterClass
and topology clusters can pass the CAPI admission webhooks.

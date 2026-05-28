# clusterctl

The provider can be packaged as a local clusterctl repository:

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

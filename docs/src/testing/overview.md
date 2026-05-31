# Testing Overview

Testing is split into layers:

- unit tests for helper functions and cloud logic
- envtest controller tests with Kubernetes API machinery
- opt-in SDK integration tests against a real STACKIT project
- opt-in e2e tests that create billable STACKIT resources

Default test command:

```sh
make test
```

Real cloud tests are gated by environment variables so they do not run
accidentally.

## Real Cloud Node Bootstrap

The real workload-cluster e2e path is intentionally allowed to use a generic
Ubuntu cloud image during development. In that mode, the e2e fixture adds
kubeadm `preKubeadmCommands` that install and configure `containerd`,
`kubelet`, `kubeadm`, and `kubectl` at runtime before kubeadm joins the node.

Treat this runtime package installation as a development fallback. It proves
the provider can pass CABPK-generated cloud-init data through STACKIT user
data, but it is slower and less deterministic than a kubeadm-ready image. A
production setup should use an image that already contains the expected
container runtime and Kubernetes node packages for the selected Kubernetes
minor.

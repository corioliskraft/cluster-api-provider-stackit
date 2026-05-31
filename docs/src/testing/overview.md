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

## Billable Workload E2E

The full workload paths create real STACKIT servers and load balancers. Run them
only in a dedicated STACKIT project with cleanup permissions and cost controls.
Each scenario is independently executable:

| Scenario | Make target | Opt-in flag |
| --- | --- | --- |
| NodeRef/providerID and Ready Nodes | `make test-e2e-workload-noderef` | `STACKIT_E2E_NODE_REF=true` |
| Worker scale with Ready Nodes | `make test-e2e-workload-scale` | `STACKIT_E2E_SCALE_WORKLOAD=true` |
| Worker upgrade with Ready replacement Nodes | `make test-e2e-workload-upgrade-workers` | `STACKIT_E2E_UPGRADE_WORKLOAD_WORKERS=true` |
| Control-plane upgrade with a Ready replacement Node | `make test-e2e-workload-upgrade-control-plane` | `STACKIT_E2E_UPGRADE_WORKLOAD_CONTROL_PLANE=true` |
| ClusterClass topology create/ready/delete | `make test-e2e-workload-topology` | `STACKIT_E2E_TOPOLOGY_WORKLOAD=true` |

Common required environment:

- `STACKIT_PROJECT_ID`
- `STACKIT_REGION`
- `STACKIT_NETWORK_ID`
- `STACKIT_IMAGE_ID`
- `STACKIT_AVAILABILITY_ZONE`
- `STACKIT_CREDENTIALS_SECRET_NAME`
- `STACKIT_CREDENTIALS_SECRET_NAMESPACE`

Useful common options:

- `KUBERNETES_VERSION`, for create/scale/topology paths
- `STACKIT_E2E_UPGRADE_FROM` and `STACKIT_E2E_UPGRADE_TO`, for upgrade paths
- `STACKIT_E2E_CNI`, `STACKIT_E2E_CNI_MANIFEST`, and CNI-specific variables
- `STACKIT_E2E_TEST_ID`, for cleanup traceability

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

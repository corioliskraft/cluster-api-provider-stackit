# Testing

## Unit and Env-Tests

Testing is split into layers:

- unit tests for helper functions and cloud logic
- envtest controller tests with Kubernetes API machinery
- opt-in SDK integration tests against a real STACKIT project
- opt-in e2e tests that create billable STACKIT resources

Default test command:

```sh
make test
```

This runs:

- generated manifest checks through controller-gen
- `go fmt`
- `go vet`
- envtest-backed Go tests, excluding `/e2e`

Real cloud tests are gated by environment variables so they do not run
accidentally.

## E2E-Tests

The e2e suite runs against an isolated kind management cluster. The generic
target runs manager startup, metrics, webhook certificate, and webhook CA
injection checks without STACKIT cloud resources:

```sh
make test-e2e
```

The real cloud paths create billable STACKIT servers, load balancers, public
IPs, and security groups. Run them only in a dedicated STACKIT project with
cleanup permissions and cost controls. The main workload scenarios have
dedicated make targets:

| Scenario | Make target | Opt-in flag |
| --- | --- | --- |
| NodeRef/providerID and Ready Nodes | `make test-e2e-workload-noderef` | `STACKIT_E2E_NODE_REF=true` |
| Bastion cluster create/delete with provider-managed bastion and node SSH security group cleanup | `make test-e2e-workload-bastion` | `STACKIT_E2E_CREATE_CLUSTER=true`, `STACKIT_E2E_BASTION=true` |
| Worker scale with Ready Nodes | `make test-e2e-workload-scale` | `STACKIT_E2E_SCALE_WORKLOAD=true` |
| Worker upgrade with Ready replacement Nodes | `make test-e2e-workload-upgrade-workers` | `STACKIT_E2E_UPGRADE_WORKLOAD_WORKERS=true` |
| Control-plane upgrade with a Ready replacement Node | `make test-e2e-workload-upgrade-control-plane` | `STACKIT_E2E_UPGRADE_WORKLOAD_CONTROL_PLANE=true` |
| ClusterClass topology create/ready/delete | `make test-e2e-workload-topology` | `STACKIT_E2E_TOPOLOGY_WORKLOAD=true` |

The suite also contains lower-level real cloud scenarios without dedicated make
targets. Run them through `go test -tags=e2e ./test/e2e` or `make test-e2e`
with the corresponding opt-in flag and a focused Ginkgo expression:

| Scenario | Opt-in flag | Suggested focus |
| --- | --- | --- |
| Single `StackitMachine` VM create/delete and leak check | `STACKIT_E2E_CREATE_VMS=true` | `create and delete a real STACKIT VM` |
| 1 control-plane / 1 worker infrastructure lifecycle without workload Node readiness | `STACKIT_E2E_CREATE_CLUSTER=true` | `create and delete a 1 control-plane / 1 worker workload Cluster` |
| Infra-only `MachineDeployment` worker VM scale up/down | `STACKIT_E2E_SCALE_WORKERS=true` | `scale a worker MachineDeployment` |
| Infra-only `MachineDeployment` worker VM replacement during version upgrade | `STACKIT_E2E_UPGRADE_WORKERS=true` | `replace worker VMs during a MachineDeployment version upgrade` |

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
- `STACKIT_SSH_KEY_NAME`, for node SSH keys when a scenario should attach one
- `STACKIT_BASTION_SSH_KEY_NAME`, or `STACKIT_SSH_KEY_NAME`, when
  `STACKIT_E2E_BASTION=true`
- `STACKIT_BASTION_ALLOWED_CIDRS`, defaulting to `0.0.0.0/0`, for bastion SSH
  ingress
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

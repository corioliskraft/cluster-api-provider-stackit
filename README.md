# cluster-api-provider-stackit

Cluster API infrastructure provider for STACKIT.

This repository contains the Kubernetes API types, controllers, cloud abstraction, and STACKIT SDK backend needed to create STACKIT infrastructure for Cluster API workload clusters. The current implementation manages existing STACKIT networks, STACKIT servers, and an optional provider-managed API server load balancer for control-plane machines.

Documentation is available as an mdBook under `docs/`:

```sh
make -C docs build
make -C docs serve
```

## Status

This provider is in early development. The controller and SDK paths have unit/envtest coverage, and real STACKIT e2e coverage exists for VM lifecycle, create/delete, and the 1 control-plane / 1 worker workload-cluster NodeRef/Node readiness path. Broader workload-cluster e2e coverage for scale, upgrades, and ClusterClass is still pending.

## Prerequisites

- Go v1.24.6 or newer
- Docker
- kubectl
- kind for local development
- clusterctl for rendering Cluster API templates
- Access to a STACKIT project
- A STACKIT service account JSON key with permissions to read networks and manage servers and load balancers
- An existing STACKIT network in the target region
- A STACKIT image ID, machine type, availability zone, and optional SSH key/security group IDs suitable for Kubernetes nodes

## Credentials

Create a Kubernetes Secret in the namespace where the workload cluster will be created. The controller expects the same Secret shape used by the STACKIT machine-controller-manager provider:

- `project-id`: STACKIT project UUID
- `serviceaccount.json`: STACKIT service account JSON key

Example:

```sh
kubectl create secret generic stackit-credentials \
  --from-literal=project-id='4cf9e1f0-1f18-4c5b-bcc5-fbd3dd6675a5' \
  --from-file=serviceaccount.json=./sa/serviceaccount.json
```

If `spec.credentialsSecretRef.namespace` is omitted on `StackitCluster`, the controller reads the Secret from the `StackitCluster` namespace.

## STACKIT Infrastructure Inputs

The provider currently creates servers and load balancers in an existing network. It does not create networks, security groups, SSH keys, or images.

Required values:

- `STACKIT_PROJECT_ID`: STACKIT project UUID
- `STACKIT_REGION`: STACKIT region, for example `eu01`
- `STACKIT_NETWORK_ID`: UUID of an existing network in the region
- `STACKIT_IMAGE_ID`: image UUID for the node operating system
- `STACKIT_MACHINE_TYPE`: STACKIT machine type, for example `c2i.2`
- `STACKIT_AVAILABILITY_ZONE`: zone, for example `eu01-1`
- `STACKIT_SSH_KEY_NAME`: existing STACKIT SSH key name, if SSH access is desired
- `STACKIT_SECURITY_GROUP_ID`: security group UUID applied to created machines

Network and security group prerequisites:

- Control-plane nodes must be reachable by the provider-managed load balancer on TCP `6443`.
- Nodes need egress for Kubernetes image pulls and package/bootstrap operations.
- Nodes in the workload cluster need the expected Kubernetes node-to-node and pod/service network traffic for the chosen CNI.
- The selected image must support cloud-init user data.

The generated provider ID format is `stackit://<server-id>`, matching `cloud-provider-stackit`.

## Cluster Template

`templates/cluster-template.yaml` is intended for `clusterctl generate cluster`.
Release assets for a local or published clusterctl repository can be generated
with `make clusterctl-release`.

Example:

```sh
export CLUSTER_NAME=capi-stackit
export NAMESPACE=default
export KUBERNETES_VERSION=v1.34.8
export CONTROL_PLANE_MACHINE_COUNT=1
export WORKER_MACHINE_COUNT=1
export STACKIT_PROJECT_ID=4cf9e1f0-1f18-4c5b-bcc5-fbd3dd6675a5
export STACKIT_REGION=eu01
export STACKIT_NETWORK_ID=3a87ac2f-8297-4dea-a9da-11d3c19e45fe
export STACKIT_IMAGE_ID=3ad2867e-695b-4ee6-9502-b563013413d4 # non-ARM Ubuntu 22.04 in eu01
export STACKIT_MACHINE_TYPE=c2i.2
export STACKIT_SSH_KEY_NAME=<ssh-key-name>
export STACKIT_CREDENTIALS_SECRET_NAME=stackit-credentials
export STACKIT_SERVICE_ACCOUNT_JSON_B64="$(base64 < ./sa/serviceaccount.json | tr -d '\n')"
export STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE=ghcr.io/stackitcloud/cloud-provider-stackit/cloud-controller-manager:v1.34.8

hack/validate-stackit-versions.sh

clusterctl generate cluster "$CLUSTER_NAME" \
  --from templates/cluster-template.yaml \
  > "${CLUSTER_NAME}.yaml"
```

The default template embeds `cloud-provider-stackit` through a Cluster API
`ClusterResourceSet`. The cloud-provider image minor must match the Kubernetes
minor:

| Kubernetes version | Required `cloud-provider-stackit` image minor |
| --- | --- |
| `v1.33.x` | `v1.33.x` |
| `v1.34.x` | `v1.34.x` |
| `v1.35.x` | `v1.35.x` |
| `v1.36.x` | `v1.36.x` |

Use a supported patch release for the selected minor. The e2e defaults currently
use Kubernetes `v1.33.12` by default and verified `cloud-provider-stackit`
image defaults `v1.33.12`, `v1.34.8`, `v1.35.3`, and `v1.36.0` for the
supported minors. Override `STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE` when using
a different patch release and keep the image minor aligned with
`KUBERNETES_VERSION`.

`clusterctl init` must run with the `ClusterResourceSet` feature enabled. The
local `hack/clusterctl-local.yaml` sets `CLUSTER_RESOURCE_SET=true`.

The default template installs `cloud-provider-stackit`, but it does not install
a CNI. Install a CNI after the workload API is reachable; the e2e NodeRef path
uses Cilium by default.

Apply the rendered manifest to a management cluster with Cluster API and this provider installed:

```sh
kubectl apply -f "${CLUSTER_NAME}.yaml"
```

## Local Development

Install CRDs into the current cluster:

```sh
make install
```

Run the controller locally against the current kubeconfig context:

```sh
make run
```

Build and deploy the controller image:

```sh
export IMG=<registry>/cluster-api-provider-stackit:<tag>
make docker-build docker-push IMG="$IMG"
make deploy IMG="$IMG"
```

For a local kind management cluster, build and load the image instead of pushing it:

```sh
export IMG=cluster-api-provider-stackit:dev
make docker-build IMG="$IMG"
kind load docker-image "$IMG" --name capi-stackit
make deploy IMG="$IMG"
```

The local development cluster used during validation is `kind-capi-stackit`.

## Tests

Run unit and envtest coverage:

```sh
make test
```

Run the opt-in STACKIT SDK integration tests with real credentials:

```sh
STACKIT_NETWORK_ID=<network-uuid> \
STACKIT_TEST_TARGET_IP=<target-ip-on-network> \
go test -tags=stackit_integration ./pkg/cloud -run 'TestSDKClient.*Integration' -v
```

The integration tests require a valid credentials Secret or local service account key, an existing network, and a target IP in that network for load balancer target-pool validation.

Integration and e2e tests are opt-in because they create or inspect real STACKIT resources:

| Test | Command | Required opt-in env vars | Common env vars | Test-specific env vars |
| --- | --- | --- | --- | --- |
| STACKIT SDK integration | `go test -tags=stackit_integration ./pkg/cloud -run 'TestSDKClient.*Integration' -v` | none | `STACKIT_PROJECT_ID`, `STACKIT_REGION`, `STACKIT_SERVICE_ACCOUNT_JSON_FILE` | `STACKIT_NETWORK_ID`, `STACKIT_TEST_TARGET_IP` |
| Real VM lifecycle e2e | `go test -tags=e2e ./test/e2e -v -ginkgo.v --ginkgo.focus='real STACKIT VM'` | `STACKIT_E2E_CREATE_VMS=true` | `KIND_CLUSTER`, `STACKIT_PROJECT_ID`, `STACKIT_REGION`, `STACKIT_NETWORK_ID`, `STACKIT_IMAGE_ID`, `STACKIT_AVAILABILITY_ZONE`, `STACKIT_CREDENTIALS_SECRET_NAME`, `STACKIT_CREDENTIALS_SECRET_NAMESPACE` | `STACKIT_E2E_TEST_ID`, `STACKIT_E2E_NAMESPACE`, `STACKIT_SSH_KEY_NAME`, `STACKIT_SECURITY_GROUP_IDS`, `STACKIT_ROOT_VOLUME_SIZE_GIB`, `STACKIT_ROOT_VOLUME_PERFORMANCE_CLASS` |
| 1 control-plane / 1 worker create-delete e2e | `go test -tags=e2e ./test/e2e -v -ginkgo.v --ginkgo.focus='1 control-plane / 1 worker'` | `STACKIT_E2E_CREATE_CLUSTER=true` | same as real VM lifecycle e2e | same as real VM lifecycle e2e |
| Workload NodeRef/providerID e2e | `go test -tags=e2e ./test/e2e -v -ginkgo.v --ginkgo.focus='align StackitMachine'` | `STACKIT_E2E_NODE_REF=true` | same as real VM lifecycle e2e | `KUBERNETES_VERSION`, `STACKIT_E2E_CNI` (`cilium` by default, or `calico`), `STACKIT_E2E_CNI_MANIFEST`, `STACKIT_E2E_CALICO_MANIFEST`, `STACKIT_E2E_CILIUM_VERSION`, `STACKIT_E2E_CILIUM_CLUSTER_POOL_IPV4_CIDR`, `STACKIT_E2E_CILIUM_CLUSTER_POOL_IPV4_MASK_SIZE`, `STACKIT_E2E_CILIUM_INSTALL_ARGS`, plus the real VM lifecycle e2e optional vars |
| Worker scale e2e | `go test -tags=e2e ./test/e2e -v -ginkgo.v --ginkgo.focus='scale a worker MachineDeployment'` | `STACKIT_E2E_SCALE_WORKERS=true` | same as real VM lifecycle e2e | `KUBERNETES_VERSION`, plus the real VM lifecycle e2e optional vars |
| Worker upgrade e2e | `go test -tags=e2e ./test/e2e -v -ginkgo.v --ginkgo.focus='replace worker VMs'` | `STACKIT_E2E_UPGRADE_WORKERS=true` | same as real VM lifecycle e2e | `STACKIT_E2E_UPGRADE_FROM`, `STACKIT_E2E_UPGRADE_TO`, plus the real VM lifecycle e2e optional vars |

The e2e tests always create STACKIT machines with machine type `c2i.2`. The NodeRef/providerID e2e installs Cilium by default using the `cilium` CLI with cluster-pool IPAM set to the workload cluster pod CIDR (`192.168.0.0/16`, mask size `24`), then waits for the Cilium Operator and DaemonSets to roll out. Set `STACKIT_E2E_CNI=calico` to use Calico instead, or set `STACKIT_E2E_CNI_MANIFEST=<url-or-path>` to apply a custom CNI manifest directly with `kubectl`.

Run the opt-in e2e test that creates and deletes a real STACKIT VM through CAPI `Cluster`/`Machine` and `StackitCluster`/`StackitMachine` objects:

```sh
export KIND_CLUSTER=capi-stackit
export STACKIT_E2E_CREATE_VMS=true
export STACKIT_PROJECT_ID=4cf9e1f0-1f18-4c5b-bcc5-fbd3dd6675a5
export STACKIT_REGION=eu01
export STACKIT_NETWORK_ID=3a87ac2f-8297-4dea-a9da-11d3c19e45fe
export STACKIT_IMAGE_ID=<image-uuid>
export STACKIT_AVAILABILITY_ZONE=eu01-1
export STACKIT_CREDENTIALS_SECRET_NAME=stackit-credentials
export STACKIT_CREDENTIALS_SECRET_NAMESPACE=default

go test -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='real STACKIT VM'
```

Optional VM e2e inputs:

- `STACKIT_E2E_NAMESPACE`: namespace for test resources, defaults to `default`
- `STACKIT_SSH_KEY_NAME`: existing SSH key name
- `STACKIT_SECURITY_GROUP_IDS`: comma-separated security group UUIDs
- `STACKIT_ROOT_VOLUME_SIZE_GIB`: defaults to `50`
- `STACKIT_ROOT_VOLUME_PERFORMANCE_CLASS`: defaults to `storage_premium_perf6`

The VM e2e test is intentionally gated by `STACKIT_E2E_CREATE_VMS=true` because it creates billable cloud resources.

## Uninstall

Delete sample or generated workload-cluster resources first:

```sh
kubectl delete -f "${CLUSTER_NAME}.yaml"
```

Undeploy the controller:

```sh
make undeploy
```

Delete CRDs:

```sh
make uninstall
```

## Distribution

Installer YAML can be generated with:

```sh
make build-installer IMG=<registry>/cluster-api-provider-stackit:<tag>
```

This writes `dist/install.yaml`, which can be published for `kubectl apply -f ...` installation. A Helm chart has not been selected as the default distribution path yet.

Clusterctl release assets can be generated with:

```sh
make clusterctl-release IMG=<registry>/cluster-api-provider-stackit:<tag>
```

This writes `infrastructure-components.yaml`, `metadata.yaml`,
`clusterclass.yaml`, `cluster-template.yaml`,
`cluster-template-development.yaml`, and `cluster-template-topology.yaml` under
`dist/clusterctl/infrastructure-stackit/v0.1.0/`, which matches clusterctl's
local repository layout.

For local validation, generate the release assets, export the STACKIT template
variables shown above, and use:

```sh
export STACKIT_CLUSTERCTL_REPOSITORY="$(pwd)/dist/clusterctl"

clusterctl init --config hack/clusterctl-local.yaml --infrastructure stackit:v0.1.0
clusterctl generate cluster stackit-test \
  --config hack/clusterctl-local.yaml \
  --infrastructure stackit:v0.1.0 \
  --kubernetes-version v1.33.12 \
  --control-plane-machine-count 1 \
  --worker-machine-count 1
```

The local clusterctl configuration enables the CAPI `ClusterTopology` feature
gate. This is required before applying `templates/clusterclass.yaml` or any
Cluster with `spec.topology`, because the CAPI and kubeadm-control-plane
webhooks reject those resources while the feature gate is disabled.
For an already initialized management cluster, either re-initialize the CAPI
providers with this config or patch the `--feature-gates` argument on the CAPI
core and kubeadm-control-plane controller manager Deployments so
`ClusterTopology=true`.

## Contributing

Before sending changes, run:

```sh
make test
```

After editing API types or kubebuilder markers, run:

```sh
make manifests
make generate
```

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

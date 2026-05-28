# cluster-api-provider-stackit

Cluster API infrastructure provider for STACKIT.

This repository contains the Kubernetes API types, controllers, cloud abstraction, and STACKIT SDK backend needed to create STACKIT infrastructure for Cluster API workload clusters. The current implementation manages existing STACKIT networks, STACKIT servers, and an optional provider-managed API server load balancer for control-plane machines.

## Status

This provider is in early development. The controller and SDK paths have unit/envtest coverage, and the SDK load balancer behavior has been validated against a real STACKIT project. Full workload-cluster e2e coverage is still pending.

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
export KUBERNETES_VERSION=v1.34.0
export CONTROL_PLANE_MACHINE_COUNT=1
export WORKER_MACHINE_COUNT=1
export STACKIT_PROJECT_ID=4cf9e1f0-1f18-4c5b-bcc5-fbd3dd6675a5
export STACKIT_REGION=eu01
export STACKIT_NETWORK_ID=3a87ac2f-8297-4dea-a9da-11d3c19e45fe
export STACKIT_IMAGE_ID=3ad2867e-695b-4ee6-9502-b563013413d4 # non-ARM Ubuntu 22.04 in eu01
export STACKIT_MACHINE_TYPE=c2i.1
export STACKIT_SSH_KEY_NAME=<ssh-key-name>
export STACKIT_CREDENTIALS_SECRET_NAME=stackit-credentials

clusterctl generate cluster "$CLUSTER_NAME" \
  --from templates/cluster-template.yaml \
  > "${CLUSTER_NAME}.yaml"
```

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

Run the opt-in e2e test that creates and deletes a real STACKIT VM through CAPI `Cluster`/`Machine` and `StackitCluster`/`StackitMachine` objects:

```sh
export KIND_CLUSTER=capi-stackit
export STACKIT_E2E_CREATE_VMS=true
export STACKIT_PROJECT_ID=4cf9e1f0-1f18-4c5b-bcc5-fbd3dd6675a5
export STACKIT_REGION=eu01
export STACKIT_NETWORK_ID=3a87ac2f-8297-4dea-a9da-11d3c19e45fe
export STACKIT_IMAGE_ID=<image-uuid>
export STACKIT_MACHINE_TYPE=c2i.2
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
  --kubernetes-version v1.31.0 \
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

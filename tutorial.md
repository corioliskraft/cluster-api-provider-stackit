# Tutorial: Using cluster-api-provider-stackit

This tutorial shows how to use this provider from a local Cluster API management cluster to create STACKIT infrastructure resources. It also includes the focused real-VM e2e test that was validated against a real STACKIT project.

The provider currently expects existing STACKIT infrastructure inputs: project, region, network, image, machine type, availability zone, and optional SSH key/security group. It creates and deletes STACKIT servers, and can manage an API server load balancer for control-plane machines.

## 1. Prerequisites

Install these tools locally:

```sh
go version
docker version
kubectl version --client
kind version
clusterctl version
stackit version
```

You also need:

- A STACKIT project.
- A STACKIT service-account JSON key.
- An existing STACKIT network in the target region.
- A node image that supports cloud-init.
- A machine type and availability zone in the same region.
- Optional existing SSH key and security group IDs.

The validated test setup used:

```sh
export STACKIT_PROJECT_ID=4cf9e1f0-1f18-4c5b-bcc5-fbd3dd6675a5
export STACKIT_REGION=eu01
export STACKIT_NETWORK_ID=3a87ac2f-8297-4dea-a9da-11d3c19e45fe
export STACKIT_IMAGE_ID=3ad2867e-695b-4ee6-9502-b563013413d4 # non-ARM Ubuntu 22.04 in eu01
export STACKIT_MACHINE_TYPE=c2i.1
export STACKIT_AVAILABILITY_ZONE=eu01-1
```

Use different values for another project or region.

## 2. Create A Management Cluster

Create or reuse an isolated kind cluster for Cluster API:

```sh
kind create cluster --name capi-stackit
kubectl config use-context kind-capi-stackit
```

Install Cluster API core, kubeadm bootstrap, and kubeadm control-plane providers:

```sh
clusterctl init \
  --core cluster-api \
  --bootstrap kubeadm \
  --control-plane kubeadm
```

Verify the providers are installed:

```sh
kubectl get pods -A | grep cluster-api
kubectl get crds | grep cluster.x-k8s.io
```

## 3. Create STACKIT Credentials

Create the credentials Secret in the namespace where workload-cluster objects will live. The expected Secret shape matches `machine-controller-manager-provider-stackit`:

- `project-id`
- `serviceaccount.json`

```sh
kubectl create secret generic stackit-credentials \
  --namespace default \
  --from-literal=project-id="${STACKIT_PROJECT_ID}" \
  --from-file=serviceaccount.json=./sa/serviceaccount.json
```

Check that the Secret exists:

```sh
kubectl get secret stackit-credentials -n default
```

## 4. Build And Deploy The Provider

Build the controller image, load it into kind, and deploy it:

```sh
export IMG=example.com/cluster-api-provider-stackit:dev

make docker-build IMG="${IMG}"
kind load docker-image "${IMG}" --name capi-stackit
make deploy IMG="${IMG}"
```

Wait for the controller manager:

```sh
kubectl rollout status \
  -n cluster-api-provider-stackit-system \
  deployment/cluster-api-provider-stackit-controller-manager
```

Confirm the CRDs are present:

```sh
kubectl get crds | grep stackit
```

## 5. Smoke Test Cluster Infrastructure

Create a minimal Cluster API `Cluster` plus `StackitCluster`. This validates credentials and network discovery without creating VMs:

```sh
cat > /tmp/stackit-infra-smoke.yaml <<EOF
apiVersion: cluster.x-k8s.io/v1beta2
kind: Cluster
metadata:
  name: stackit-smoke
  namespace: default
spec:
  infrastructureRef:
    apiGroup: infrastructure.cluster.x-k8s.io
    kind: StackitCluster
    name: stackit-smoke
  clusterNetwork:
    pods:
      cidrBlocks:
        - 192.168.0.0/16
    services:
      cidrBlocks:
        - 10.128.0.0/12
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitCluster
metadata:
  name: stackit-smoke
  namespace: default
spec:
  projectID: ${STACKIT_PROJECT_ID}
  region: ${STACKIT_REGION}
  credentialsSecretRef:
    name: stackit-credentials
    namespace: default
  network:
    id: ${STACKIT_NETWORK_ID}
  apiServerLoadBalancer:
    enabled: false
  controlPlaneEndpoint:
    host: 203.0.113.10
    port: 6443
EOF

kubectl apply -f /tmp/stackit-infra-smoke.yaml
```

Wait for readiness:

```sh
kubectl wait \
  --for=jsonpath='{.status.ready}'=true \
  stackitcluster/stackit-smoke \
  -n default \
  --timeout=180s

kubectl get stackitcluster stackit-smoke -n default -o yaml
kubectl get cluster stackit-smoke -n default
```

Clean up the smoke resources:

```sh
kubectl delete -f /tmp/stackit-infra-smoke.yaml
```

## 6. Render A Workload Cluster

The repository includes `templates/cluster-template.yaml` for `clusterctl generate cluster`.

Set the template variables:

```sh
export CLUSTER_NAME=stackit-workload
export NAMESPACE=default
export KUBERNETES_VERSION=v1.34.0
export CONTROL_PLANE_MACHINE_COUNT=1
export WORKER_MACHINE_COUNT=1
export STACKIT_SSH_KEY_NAME=<ssh-key-name>
export STACKIT_SECURITY_GROUP_ID=<security-group-uuid>
```

Render the manifest:

```sh
clusterctl generate cluster "${CLUSTER_NAME}" \
  --from templates/cluster-template.yaml \
  > "${CLUSTER_NAME}.yaml"
```

Review it before applying:

```sh
kubectl apply --dry-run=server -f "${CLUSTER_NAME}.yaml"
```

Apply it when ready:

```sh
kubectl apply -f "${CLUSTER_NAME}.yaml"
```

Watch the CAPI and provider resources:

```sh
kubectl get cluster,machine -n default
kubectl get stackitcluster,stackitmachine -n default
kubectl get kubeadmcontrolplane,machinedeployment -n default
```

Useful detail commands:

```sh
kubectl describe stackitcluster "${CLUSTER_NAME}" -n default
kubectl describe stackitmachine -n default
kubectl logs \
  -n cluster-api-provider-stackit-system \
  deployment/cluster-api-provider-stackit-controller-manager \
  -c manager \
  --tail=200
```

After machines are provisioned, provider IDs should use the STACKIT cloud-provider-compatible format:

```text
stackit://<server-id>
```

## 7. Validate VM Create/Delete With The E2E Test

For a focused validation that creates exactly one STACKIT VM and then deletes it, use the opt-in e2e scenario. This is the safest first real-VM path because the test verifies cleanup through the STACKIT SDK.

```sh
export KIND_CLUSTER=capi-stackit
export STACKIT_E2E_CREATE_VMS=true
export STACKIT_CREDENTIALS_SECRET_NAME=stackit-credentials
export STACKIT_CREDENTIALS_SECRET_NAMESPACE=default
export STACKIT_ROOT_VOLUME_SIZE_GIB=10

go test -tags=e2e ./test/e2e -v -ginkgo.v \
  --ginkgo.focus='real STACKIT VM'
```

Expected behavior:

1. The test builds and deploys the local controller image into `kind-capi-stackit`.
2. It applies a CAPI `Cluster`/`Machine` plus `StackitCluster`/`StackitMachine`.
3. The provider creates one STACKIT server using the configured Ubuntu 22.04 image.
4. The test reads `StackitMachine.status.instanceID`.
5. The test verifies the server exists through the STACKIT SDK.
6. The test deletes the `StackitMachine`.
7. The provider deletes the STACKIT server.
8. The test verifies the server is no longer present.

Confirm no servers remain:

```sh
stackit server list \
  --project-id "${STACKIT_PROJECT_ID}" \
  --region "${STACKIT_REGION}"
```

## 8. Cleanup

Delete a generated workload cluster:

```sh
kubectl delete -f "${CLUSTER_NAME}.yaml"
```

Watch until provider resources are gone:

```sh
kubectl get cluster,machine,stackitcluster,stackitmachine -A
```

Undeploy this provider:

```sh
make undeploy
```

Remove provider CRDs:

```sh
make uninstall
```

Delete the kind management cluster when you no longer need it:

```sh
kind delete cluster --name capi-stackit
```

## Troubleshooting

If `StackitCluster` does not become ready:

```sh
kubectl describe stackitcluster <name> -n <namespace>
kubectl get secret stackit-credentials -n <namespace>
stackit network describe "${STACKIT_NETWORK_ID}" \
  --project-id "${STACKIT_PROJECT_ID}" \
  --region "${STACKIT_REGION}"
```

If `StackitMachine` does not become ready:

```sh
kubectl describe stackitmachine <name> -n <namespace>
kubectl logs \
  -n cluster-api-provider-stackit-system \
  deployment/cluster-api-provider-stackit-controller-manager \
  -c manager \
  --tail=200
stackit image list \
  --project-id "${STACKIT_PROJECT_ID}" \
  --region "${STACKIT_REGION}"
stackit server machine-type list \
  --project-id "${STACKIT_PROJECT_ID}" \
  --region "${STACKIT_REGION}"
```

Common issues:

- Credentials Secret is missing `project-id` or `serviceaccount.json`.
- The service account lacks permissions for IAAS or load balancer APIs.
- The network ID does not exist in the selected region.
- The image architecture does not match the selected machine type.
- A boot volume create request must use `bootVolume.source` for the image and must not also send top-level `imageId`; this provider handles that path.
- Security groups block Kubernetes API server traffic on TCP `6443`.
- Nodes lack egress required by cloud-init, kubeadm, image pulls, or the chosen CNI.

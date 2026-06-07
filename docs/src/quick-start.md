# Tutorial: cluster-api-provider-stackit

Minimal path: run Cluster API in kind, deploy this provider, then create one
STACKIT workload cluster. Workload VMs are real STACKIT resources.

## Prerequisites

Install `go`, `docker`, `kubectl`, `kind`, `clusterctl`, and the `stackit` CLI.
You also need a STACKIT project, service-account JSON key, existing network,
image, and machine type.

```sh
export STACKIT_PROJECT_ID=<project-uuid>
export STACKIT_REGION=eu01
export STACKIT_NETWORK_ID=<network-uuid>
export STACKIT_IMAGE_ID=<image-uuid>
export STACKIT_MACHINE_TYPE=c2i.2
export STACKIT_SERVICE_ACCOUNT_JSON_FILE=./.stackit/cluster-api-provider-stackit.json
export STACKIT_SERVICE_ACCOUNT_JSON_B64="$(base64 < "${STACKIT_SERVICE_ACCOUNT_JSON_FILE}" | tr -d '\n')"
export STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE=ghcr.io/stackitcloud/cloud-provider-stackit/cloud-controller-manager:v1.35.3
```

The default template does not configure SSH access. To use an SSH key, add
`sshKeyName: <key-name>` to the `StackitMachineTemplate` specs before creating
the workload cluster.

## Create the management cluster

```sh
kind create cluster --name capi-stackit
kubectl config use-context kind-capi-stackit

clusterctl init \
  --config hack/clusterctl-local.yaml \
  --core cluster-api \
  --bootstrap kubeadm \
  --control-plane kubeadm
```

 ## Install credentials and provider
 
 ```sh
 kubectl create secret generic stackit-credentials \
   --namespace default \
   --from-literal=project-id="${STACKIT_PROJECT_ID}" \
   --from-file=serviceaccount.json="${STACKIT_SERVICE_ACCOUNT_JSON_FILE}"
 
 export IMG=cluster-api-provider-stackit:dev
 make docker-build IMG="${IMG}"
 kind load docker-image "${IMG}" --name capi-stackit
 make deploy IMG="${IMG}"
 
 kubectl rollout status \
   -n cluster-api-provider-stackit-system \
   deployment/cluster-api-provider-stackit-controller-manager
 ```
 
 ## Create a workload cluster
 
 ```sh
 export CLUSTER_NAME=stackit-workload
 export NAMESPACE=default
 export KUBERNETES_VERSION=v1.35.3
 export KUBERNETES_APT_REPOSITORY_MINOR=v1.35
 export CONTROL_PLANE_MACHINE_COUNT=1
 export WORKER_MACHINE_COUNT=1
 export STACKIT_CREDENTIALS_SECRET_NAME=stackit-credentials
 
 clusterctl generate cluster "${CLUSTER_NAME}" \
   --from templates/cluster-template.yaml \
   --target-namespace "${NAMESPACE}" \
   | kubectl apply -f -
 ```
 
 Watch progress:
 
 ```sh
 kubectl get cluster,machine,stackitcluster,stackitmachine -n "${NAMESPACE}"
 kubectl logs \
   -n cluster-api-provider-stackit-system \
   deployment/cluster-api-provider-stackit-controller-manager \
   -c manager \
   --tail=100
 ```
 
 After the workload API is reachable, install a CNI:
 
 ```sh
 clusterctl get kubeconfig "${CLUSTER_NAME}" -n "${NAMESPACE}" > /tmp/"${CLUSTER_NAME}".kubeconfig
 make install-workload-cni WORKLOAD_KUBECONFIG=/tmp/"${CLUSTER_NAME}".kubeconfig
 ```
 
 ## Clean up
 
 ```sh
 kubectl delete cluster "${CLUSTER_NAME}" -n "${NAMESPACE}"
 kubectl get cluster,machine,stackitcluster,stackitmachine -A
 
 make undeploy
 make uninstall
 kind delete cluster --name capi-stackit
 ```
 
 If deletion is interrupted, inspect remaining cloud resources:
 
 ```sh
 stackit server list --project-id "${STACKIT_PROJECT_ID}" --region "${STACKIT_REGION}"
 stackit load-balancer list --project-id "${STACKIT_PROJECT_ID}" --region "${STACKIT_REGION}"
```

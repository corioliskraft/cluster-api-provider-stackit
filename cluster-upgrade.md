# Cluster Upgrade Tutorial

This tutorial shows how to create a STACKIT workload cluster with Kubernetes
`v1.31.0` and upgrade it to `v1.32.0` with Cluster API.

## Prerequisites

You need a management cluster with Cluster API and the STACKIT infrastructure
provider installed.

The workload cluster also needs a credentials Secret in the target namespace:

```sh
export STACKIT_SERVICE_ACCOUNT_JSON_FILE=./.stackit/serviceaccount.json

kubectl create secret generic stackit-credentials \
  --namespace default \
  --from-file=serviceaccount.json="${STACKIT_SERVICE_ACCOUNT_JSON_FILE}"
```

Export the template variables used by `templates/cluster-template.yaml`:

```sh
export CLUSTER_NAME=stackit-upgrade
export NAMESPACE=default
export KUBERNETES_VERSION=v1.31.0
export CONTROL_PLANE_MACHINE_COUNT=1
export WORKER_MACHINE_COUNT=1

export STACKIT_PROJECT_ID=<project-uuid>
export STACKIT_REGION=eu01
export STACKIT_NETWORK_ID=<network-uuid>
export STACKIT_IMAGE_ID=<ubuntu-image-uuid>
export STACKIT_MACHINE_TYPE=c2i.1
export STACKIT_SSH_KEY_NAME=<ssh-key-name>
export STACKIT_CREDENTIALS_SECRET_NAME=stackit-credentials
```

## Create The v1.31 Cluster

Render the Cluster API objects:

```sh
clusterctl generate cluster "$CLUSTER_NAME" \
  --from templates/cluster-template.yaml \
  > "${CLUSTER_NAME}.yaml"
```

Apply them to the management cluster:

```sh
kubectl apply -f "${CLUSTER_NAME}.yaml"
```

Watch the Cluster API resources:

```sh
kubectl get cluster,kubeadmcontrolplane,machinedeployment,machine \
  -n "$NAMESPACE" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}" \
  --watch
```

Check the STACKIT infrastructure resources:

```sh
kubectl get stackitcluster,stackitmachine -n "$NAMESPACE" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}"
```

Before upgrading, verify that the cluster is on `v1.31.0`:

```sh
kubectl get kubeadmcontrolplane "${CLUSTER_NAME}-control-plane" \
  -n "$NAMESPACE" \
  -o jsonpath='{.spec.version}{"\n"}'

kubectl get machinedeployment "${CLUSTER_NAME}-md-0" \
  -n "$NAMESPACE" \
  -o jsonpath='{.spec.template.spec.version}{"\n"}'
```

Both commands should print:

```text
v1.31.0
```

## Upgrade To v1.32

Set the target version:

```sh
export UPGRADE_VERSION=v1.32.0
```

Upgrade the control plane first:

```sh
kubectl patch kubeadmcontrolplane "${CLUSTER_NAME}-control-plane" \
  -n "$NAMESPACE" \
  --type merge \
  -p "{\"spec\":{\"version\":\"${UPGRADE_VERSION}\"}}"
```

Watch the control plane rollout:

```sh
kubectl get kubeadmcontrolplane,machine \
  -n "$NAMESPACE" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}" \
  --watch
```

Wait until the `KubeadmControlPlane` reports the desired version and all control
plane Machines are up to date:

```sh
kubectl get kubeadmcontrolplane "${CLUSTER_NAME}-control-plane" \
  -n "$NAMESPACE" \
  -o wide
```

Then upgrade the workers:

```sh
kubectl patch machinedeployment "${CLUSTER_NAME}-md-0" \
  -n "$NAMESPACE" \
  --type merge \
  -p "{\"spec\":{\"template\":{\"spec\":{\"version\":\"${UPGRADE_VERSION}\"}}}}"
```

Watch the worker rollout:

```sh
kubectl get machinedeployment,machineset,machine \
  -n "$NAMESPACE" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}" \
  --watch
```

The `MachineDeployment` should create a new MachineSet for `v1.32.0`, create
replacement worker Machines, and delete the old `v1.31.0` worker Machines after
the replacements become available.

## Verify The Upgrade

Check that the desired versions are updated:

```sh
kubectl get kubeadmcontrolplane "${CLUSTER_NAME}-control-plane" \
  -n "$NAMESPACE" \
  -o jsonpath='{.spec.version}{"\n"}'

kubectl get machinedeployment "${CLUSTER_NAME}-md-0" \
  -n "$NAMESPACE" \
  -o jsonpath='{.spec.template.spec.version}{"\n"}'
```

Both should now print:

```text
v1.32.0
```

Verify that Machines have provider IDs:

```sh
kubectl get machine -n "$NAMESPACE" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}" \
  -o custom-columns=NAME:.metadata.name,VERSION:.spec.version,READY:.status.conditions[-1].status,PROVIDERID:.spec.providerID
```

Provider IDs should use the STACKIT format:

```text
stackit://<server-id>
```

If you have workload-cluster kubeconfig access, verify the Nodes too:

```sh
kubectl --kubeconfig "${CLUSTER_NAME}.kubeconfig" get nodes -o wide
```

All Nodes should report `v1.32.0`.

## Verify Old VMs Are Gone

After the rollout completes, old Machines should be deleted from Kubernetes:

```sh
kubectl get machine -n "$NAMESPACE" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}"
```

Only the replacement Machines should remain.

Check the corresponding infrastructure resources:

```sh
kubectl get stackitmachine -n "$NAMESPACE" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}"
```

Each remaining `StackitMachine` should be ready and should have an
`instanceID`/`providerID`. Deleted Machines should no longer have matching
STACKIT VMs.

## Rollback

If the upgrade is stuck before old Machines are removed, inspect conditions:

```sh
kubectl describe kubeadmcontrolplane "${CLUSTER_NAME}-control-plane" -n "$NAMESPACE"
kubectl describe machinedeployment "${CLUSTER_NAME}-md-0" -n "$NAMESPACE"
kubectl get machine -n "$NAMESPACE" -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}" -o wide
```

To roll back the desired version, patch the resources back to `v1.31.0`:

```sh
kubectl patch kubeadmcontrolplane "${CLUSTER_NAME}-control-plane" \
  -n "$NAMESPACE" \
  --type merge \
  -p '{"spec":{"version":"v1.31.0"}}'

kubectl patch machinedeployment "${CLUSTER_NAME}-md-0" \
  -n "$NAMESPACE" \
  --type merge \
  -p '{"spec":{"template":{"spec":{"version":"v1.31.0"}}}}'
```

## Cleanup

Delete the workload cluster:

```sh
kubectl delete cluster "$CLUSTER_NAME" -n "$NAMESPACE" --wait=true --timeout=45m
```

Verify no provider resources remain:

```sh
kubectl get stackitcluster,stackitmachine -n "$NAMESPACE" \
  -l "cluster.x-k8s.io/cluster-name=${CLUSTER_NAME}"
```

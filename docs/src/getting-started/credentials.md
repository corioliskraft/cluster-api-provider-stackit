# Credentials

Create a Secret in the namespace where workload-cluster objects will be created.
The Secret shape matches the STACKIT machine-controller-manager provider:

- `project-id`
- `serviceaccount.json`

Example:

```sh
kubectl create secret generic stackit-credentials \
  --namespace default \
  --from-literal=project-id="${STACKIT_PROJECT_ID}" \
  --from-file=serviceaccount.json=./sa/serviceaccount.json
```

If `StackitCluster.spec.credentialsSecretRef.namespace` is omitted, the
controller reads the Secret from the `StackitCluster` namespace.

Do not commit service-account files or generated kubeconfigs. Local credential
files should stay under ignored paths such as `sa/`.

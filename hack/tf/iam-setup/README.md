# IAM setup for cluster-api-provider-stackit

This OpenTofu setup creates:

- a strict custom project role named `cluster-api-provider-stackit`
- a dedicated STACKIT service account
- a project role assignment from the strict role to the service account
- a service-account key JSON for the Kubernetes `stackit-credentials` Secret

The runtime role contains only the permissions used by the current provider
controllers.

## Bootstrap permissions

The bootstrap service account used for this OpenTofu setup needs:

- `iam.role.add`
- `iam.role.get`
- `iam.role.list`
- `iam.member.add`
- `iam.member.get`
- `iam.service-account.create`
- `iam.service-account.get`
- `iam.service-account.list`
- `iam.service-account-key.create`

For `tofu destroy`, it also needs:

- `iam.role.remove`
- `iam.service-account.delete`
- `iam.service-account-key.delete`

## Usage

Copy the example variables file and fill in the project ID plus bootstrap key
path:

```sh
cd hack/tf/iam-setup
cp terraform.tfvars.example terraform.tfvars
```

Apply:

```sh
tofu init
tofu apply
```

Write the generated provider key to a local file:

```sh
export STACKIT_SERVICE_ACCOUNT_JSON_FILE=../../../.stackit/cluster-api-provider-stackit-serviceaccount.json

mkdir -p "$(dirname "${STACKIT_SERVICE_ACCOUNT_JSON_FILE}")"
tofu output -raw service_account_key_json > "${STACKIT_SERVICE_ACCOUNT_JSON_FILE}"
```

Create the Kubernetes Secret in the namespace that contains your
`StackitCluster` objects:

```sh
kubectl create secret generic stackit-credentials \
  --namespace default \
  --from-literal=project-id="$(tofu output -raw project_id 2>/dev/null || grep '^project_id' terraform.tfvars | cut -d= -f2 | tr -d ' \"')" \
  --from-file=serviceaccount.json="${STACKIT_SERVICE_ACCOUNT_JSON_FILE}"
```

The generated key is also stored in OpenTofu state as a sensitive value. Protect
the state file like a credential secret.

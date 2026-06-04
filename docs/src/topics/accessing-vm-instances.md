# Accessing cluster instances

By default, workload clusters created by `cluster-api-provider-stackit` do not
expose SSH access. Cluster API bootstrap does not require SSH; CABPK provides
cloud-init user data through Kubernetes bootstrap Secrets.

For break-glass access, a `StackitCluster` can optionally ask the provider to
create one managed SSH bastion VM. The bastion is attached to the same STACKIT
network as the cluster nodes, gets a provider-managed public IP, and has a
provider-managed security group allowing TCP/22 only from configured CIDRs.

## Prerequisites

- The workload cluster was generated with `clusterctl generate cluster` and
  applied to the management cluster.
- The configured STACKIT service account role includes the bastion permissions
  documented in [IAM Permissions Used](./iam-permissions.md).
- The bastion image is an Ubuntu image with SSH enabled for the expected user,
  normally `ubuntu`.
- The STACKIT SSH key named in `spec.bastion.sshKeyName` already exists for the
  service account used by the provider. STACKIT key pairs are not shared across
  service accounts, so importing a key with an inspection or admin service
  account does not make it usable by the controller service account.
- To SSH from the bastion into cluster nodes, the control-plane and worker
  `StackitMachineTemplate` resources must also set
  `spec.template.spec.sshKeyName`. The bastion does not inject SSH keys into
  existing node VMs.

## Enable the bastion

Patch or edit the generated `StackitCluster`:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitCluster
metadata:
  name: <cluster-name>
  namespace: <namespace>
spec:
  bastion:
    enabled: true
    imageID: <ubuntu-image-id>
    machineType: <machine-type>
    sshKeyName: <existing-stackit-ssh-key-name>
    allowedCIDRs:
      - <your-public-ip-or-network-cidr>
    rootVolume:
      sizeGiB: 50
      performanceClass: storage_premium_perf6
      deleteOnTermination: true
```

Use a narrow `allowedCIDRs` value such as `203.0.113.10/32` where possible.
`0.0.0.0/0` allows SSH from anywhere and should only be used deliberately.
Set `rootVolume` when the chosen machine type's flavor disk is too small for
the chosen image.

For node access, set the same or another existing SSH key on the machine
templates:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitMachineTemplate
metadata:
  name: <machine-template-name>
spec:
  template:
    spec:
      sshKeyName: <existing-stackit-ssh-key-name>
```

Leaving `StackitMachineTemplate.spec.template.spec.sshKeyName` empty is valid,
but SSH to nodes through the bastion will not work.

## Get the bastion IP

Use the management-cluster kubeconfig:

```sh
kubectl get stackitcluster <cluster-name> \
  --namespace <namespace> \
  -o jsonpath='{.status.bastion.publicIP}'
```

Or inspect the printcolumn:

```sh
kubectl get stackitclusters --namespace <namespace>
```

The `Bastion IP` column is empty until STACKIT has assigned and attached the
public IP.

## Get node internal IPs

Use the workload-cluster kubeconfig:

```sh
kubectl get nodes \
  -o custom-columns=NAME:.metadata.name,IP:'{.status.addresses[?(@.type=="InternalIP")].address}'
```

Alternatively, inspect the management-cluster `StackitMachine` addresses:

```sh
kubectl get stackitmachines --namespace <namespace> \
  -o custom-columns=NAME:.metadata.name,IP:'{.status.addresses[?(@.type=="InternalIP")].address}'
```

## Connect through the bastion

Set local variables:

```sh
export CLUSTER_SSH_KEY="$HOME/.ssh/<private-key-file>"
export BASTION_HOST="$(kubectl get stackitcluster <cluster-name> \
  --namespace <namespace> \
  -o jsonpath='{.status.bastion.publicIP}')"
```

Connect to a node internal IP through the bastion:

```sh
ssh -i "${CLUSTER_SSH_KEY}" ubuntu@<node-internal-ip> \
  -o "ProxyCommand ssh -W %h:%p -i ${CLUSTER_SSH_KEY} ubuntu@${BASTION_HOST}"
```

An equivalent SSH config entry:

```text
Host 10.*
  User ubuntu
  IdentityFile <cluster-ssh-key>
  ProxyCommand ssh -W %h:%p ubuntu@<bastion-public-ip>
```

## Cleanup

The provider deletes the managed bastion server, public IP, security group, and
security group rules when the `StackitCluster` is deleted. It also deletes
previously recorded managed bastion resources when `spec.bastion.enabled` is
changed from `true` to `false`.

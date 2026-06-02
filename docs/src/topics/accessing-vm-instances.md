# Accessing cluster instances

## Overview

After running `clusterctl generate cluster` to generate the configuration for a new workload cluster (and then redirecting that output to a file for use with `kubectl apply`, or piping it directly to `kubectl apply`), the new workload cluster will be deployed. This document explains how to access the new workload cluster's nodes.

## Prerequisites

1. `clusterctl generate cluster` was successfully executed to generate the configuration for a new workload cluster
2. The configuration for the new workload cluster was applied to the management cluster using `kubectl apply` and the cluster is up and running in an STACKIT environment.
3. The SSH key referenced by `clusterctl` in step 1 exists in STACKIT and is stored in the correct location locally for use by SSH (on macOS/Linux systems, this is typically `$HOME/.ssh`). This document will refer to this key as `cluster-api-provider-stackit.sigs.k8s.io`.

## Accessing nodes via SSH

By default, workload clusters created in STACKIT will _not_ support access via SSH. However, the manifest for a workload cluster can be modified to include an SSH bastion host, created and managed by the management cluster, to enable SSH access to cluster nodes. The bastion node is created in a public subnet and provides SSH access from the world. It runs the official Ubuntu Linux image.

### Enabling the bastion host

To configure the Cluster API Provider for STACKIT to create an SSH bastion host, add this line to the STACKITCluster spec:

```yaml
spec:
  bastion:
    enabled: true
    ami: "3ad2867e-695b-4ee6-9502-b563013413d4"
```

The image idea must refer to a Ubuntu image.

#### Obtain public IP address of the bastion node

Once the workload cluster is up and running after being configured for an SSH bastion host, you can use the `kubectl get stackitcluster` command to look up the public IP address of the bastion host (make sure the `kubectl` context is set to the management cluster). The output will look something like this:

```bash
NAME   CLUSTER   READY   VPC                     BASTION IP
test   test      true    vpc-1739285ed052be7ad   1.2.3.4
```

#### Setting up the SSH key path

Assumming that the `cluster-api-provider-stackit.sigs.k8s.io` SSH key is stored in
`$HOME/.ssh/cluster-api-provider-stackit`, use this command to set up an environment variable for use in a later command:

```bash
export CLUSTER_SSH_KEY=$HOME/.ssh/cluster-api-provider-stackit
```

#### Get private IP addresses of nodes in the cluster

To get the private IP addresses of nodes in the cluster (nodes may be control plane nodes or worker nodes), use this `kubectl` command with the context set to the management cluster:

```bash
kubectl get nodes -o custom-columns=NAME:.metadata.name,\
IP:"{.status.addresses[?(@.type=='InternalIP')].address}"
```

This will produce output that looks like this:

```bash
NAME                                         IP
ip-10-0-0-16.us-west-2.compute.internal   10.0.0.16
ip-10-0-0-68.us-west-2.compute.internal   10.0.0.68
```

The above command returns IP addresses of the nodes in the cluster. In this
case, the values returned are `10.0.0.16` and `10.0.0.68`.

### Connecting to the nodes via SSH

To access one of the nodes (either a control plane node or a worker node) via the SSH bastion host, use this command:

```bash
ssh -i ${CLUSTER_SSH_KEY} ubuntu@<NODE_IP> \
	-o "ProxyCommand ssh -W %h:%p -i ${CLUSTER_SSH_KEY} ubuntu@${BASTION_HOST}"
```

And use this command if you are using a EKS based cluster:

```bash
ssh -i ${CLUSTER_SSH_KEY} ec2-user@<NODE_IP> \
	-o "ProxyCommand ssh -W %h:%p -i ${CLUSTER_SSH_KEY} ubuntu@${BASTION_HOST}"
```


If the whole document is followed, the value of `<NODE_IP>` will be either
10.0.0.16 or 10.0.0.68.

Alternately, users can add a configuration stanza to their SSH configuration file (typically found on macOS/Linux systems as `$HOME/.ssh/config`):

```text
Host 10.0.*
  User ubuntu
  IdentityFile <CLUSTER_SSH_KEY>
  ProxyCommand ssh -W %h:%p ubuntu@<BASTION_HOST>
```

## Additional Notes

### Using the STACKIT CLI instead of `kubectl`

It is also possible to use STACKIT CLI commands instead of `kubectl` to gather information about the cluster nodes.

For example, to use the STACKIT CLI to get the public IP address of the SSH bastion host, use this STACKIT CLI command:

```bash
export BASTION_HOST=$(<insert command>)
```

You should substitute the correct cluster name for `<CLUSTER_NAME>` in the above command. (**NOTE**: If `make manifests` was used to generate manifests, by default the `<CLUSTER_NAME>` is set to `test1`.)

Similarly, to obtain the list of private IP addresses of the cluster nodes, use this STACKIT CLI command:

```bash
<insert command>
```

Note that your STACKIT CLI must be configured with credentials that enable you to query the STACKIT API.

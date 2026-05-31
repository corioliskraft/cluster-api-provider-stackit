# STACKIT Cloud Resources

The provider currently consumes existing STACKIT resources. It creates and
deletes workload-cluster servers and, when enabled, an API server load balancer,
but it does not create networks, security groups, SSH keys, or images.

## Network

`StackitCluster.spec.network.id` is the STACKIT network ID used by all cluster
machines and the provider-managed API server load balancer.

Choose or create a network in the same region as the workload cluster. The
network must provide:

- IP addresses for all control-plane and worker machines.
- Routing between workload-cluster nodes.
- Egress for cloud-init, package repositories when using development images,
  container image pulls, DNS, NTP, and STACKIT APIs.
- Traffic required by the selected CNI.

Network creation is intentionally outside the provider for now because networks
usually encode platform-level policy such as address planning, routing,
peering, VPN, DNS, and shared-service access.

## Security Groups

`StackitMachine.spec.securityGroups` is an optional list of STACKIT security
group IDs attached to each created server. If omitted, STACKIT project defaults
apply.

For production clusters, prefer explicit security groups over implicit project
defaults. At minimum, the selected security groups must allow:

- The Kubernetes API server on TCP `6443` from the API server load balancer and
  from trusted administration networks.
- Node-to-node traffic required by kubeadm, kubelet, the control plane, and the
  selected CNI.
- Egress for bootstrap, package repositories if needed, image registries, DNS,
  NTP, and STACKIT APIs.

Do not open SSH ingress by default. If SSH is needed for break-glass debugging,
restrict it to trusted CIDRs.

## SSH Keys

`StackitMachine.spec.sshKeyName` is optional. It references an existing STACKIT
SSH key pair by name.

SSH is not used for Cluster API bootstrap. Kubeadm bootstrap data is generated
by CABPK, stored in a Kubernetes Secret, and sent to STACKIT as cloud-init user
data when each server is created. Leave `sshKeyName` empty when SSH access is
not required.

## Node Images

`StackitMachine.spec.imageID` is the STACKIT image ID used for the node root
disk. The image must be compatible with cloud-init and kubeadm bootstrap.

For production clusters, use kubeadm-ready images that already contain the
expected container runtime and Kubernetes node packages for the chosen
Kubernetes minor. Runtime package installation is useful for development, but
it adds dependency on external package repositories during cluster creation and
slows down Machine readiness.

The current real e2e path can bootstrap generic Ubuntu images by installing
`containerd`, `kubelet`, `kubeadm`, and `kubectl` in kubeadm
`preKubeadmCommands`. Treat this as a development fallback, not the desired
production path.

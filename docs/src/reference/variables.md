# Template Variables

Common clusterctl variables:

| Variable | Description |
| --- | --- |
| `CLUSTER_NAME` | Workload Cluster name |
| `NAMESPACE` | Namespace for Cluster API resources |
| `KUBERNETES_VERSION` | Kubernetes version, for example `v1.33.12`. Supported workload cluster minors are v1.33.x through v1.36.x; use the latest patch release for the selected minor |
| `CONTROL_PLANE_MACHINE_COUNT` | Control-plane replica count |
| `WORKER_MACHINE_COUNT` | Worker replica count |
| `STACKIT_PROJECT_ID` | STACKIT project UUID |
| `STACKIT_REGION` | STACKIT region, for example `eu01` |
| `STACKIT_NETWORK_ID` | Existing STACKIT network UUID |
| `STACKIT_IMAGE_ID` | Node image UUID |
| `STACKIT_MACHINE_TYPE` | STACKIT machine type |
| `STACKIT_SSH_KEY_NAME` | Optional existing SSH key name. Leave empty when SSH access is not required; SSH is not used for Cluster API bootstrap |
| `STACKIT_CREDENTIALS_SECRET_NAME` | Kubernetes Secret with STACKIT credentials |
| `STACKIT_SERVICE_ACCOUNT_JSON_B64` | Base64-encoded STACKIT service account JSON for the workload-cluster cloud controller manager Secret |
| `STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE` | `cloud-provider-stackit` image. The image tag minor must match `KUBERNETES_VERSION` |
| `KUBERNETES_APT_REPOSITORY_MINOR` | Kubernetes package repository minor used by topology development fallback `preKubeadmCommands`, for example `v1.35` |
| `CLUSTER_CLASS_NAMESPACE` | Namespace containing the ClusterClass |

Optional bastion variables may be used when patching a generated
`StackitCluster` to enable SSH bastion access:

| Variable | Description |
| --- | --- |
| `STACKIT_BASTION_ENABLED` | Set to `true` to enable the provider-managed SSH bastion |
| `STACKIT_BASTION_IMAGE_ID` | STACKIT image UUID for the bastion VM |
| `STACKIT_BASTION_MACHINE_TYPE` | STACKIT machine type for the bastion VM |
| `STACKIT_BASTION_SSH_KEY_NAME` | Existing STACKIT SSH key name attached to the bastion VM |
| `STACKIT_BASTION_ALLOWED_CIDRS` | CIDR allowlist for bastion TCP/22 access, for example `203.0.113.10/32` |

Optional workload CNI helper variables:

| Variable | Description |
| --- | --- |
| `WORKLOAD_KUBECONFIG` | Path to the workload-cluster kubeconfig used by `make install-workload-cni` |
| `STACKIT_WORKLOAD_CNI` | CNI to install with `make install-workload-cni`; supported values are `cilium` and `calico` |
| `CILIUM_VERSION` | Cilium version used by the helper, defaults to `1.19.4` |
| `CILIUM_VALUES` | Helm values file for Cilium, defaults to `templates/addons/cilium-values.yaml` |
| `CALICO_MANIFEST` | Calico manifest URL or path for the helper |
| `CNI_MANIFEST` | Custom CNI manifest URL or path; when set, it takes precedence over `STACKIT_WORKLOAD_CNI` |

Development template variables may also include:

| Variable | Description |
| --- | --- |
| `STACKIT_AVAILABILITY_ZONE` | STACKIT availability zone |
| `STACKIT_SECURITY_GROUP_ID` | Security group UUID |

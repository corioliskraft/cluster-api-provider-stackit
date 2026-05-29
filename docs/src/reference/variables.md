# Template Variables

Common clusterctl variables:

| Variable | Description |
| --- | --- |
| `CLUSTER_NAME` | Workload Cluster name |
| `NAMESPACE` | Namespace for Cluster API resources |
| `KUBERNETES_VERSION` | Kubernetes version, for example `v1.33.0`. Supported workload cluster minors are v1.33.x through v1.36.x |
| `CONTROL_PLANE_MACHINE_COUNT` | Control-plane replica count |
| `WORKER_MACHINE_COUNT` | Worker replica count |
| `STACKIT_PROJECT_ID` | STACKIT project UUID |
| `STACKIT_REGION` | STACKIT region, for example `eu01` |
| `STACKIT_NETWORK_ID` | Existing STACKIT network UUID |
| `STACKIT_IMAGE_ID` | Node image UUID |
| `STACKIT_MACHINE_TYPE` | STACKIT machine type |
| `STACKIT_SSH_KEY_NAME` | Optional existing SSH key name |
| `STACKIT_CREDENTIALS_SECRET_NAME` | Kubernetes Secret with STACKIT credentials |
| `STACKIT_SERVICE_ACCOUNT_JSON_B64` | Base64-encoded STACKIT service account JSON for the workload-cluster cloud controller manager Secret |
| `STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE` | `cloud-provider-stackit` image. The image tag minor must match `KUBERNETES_VERSION` |
| `CLUSTER_CLASS_NAMESPACE` | Namespace containing the ClusterClass |

Development template variables may also include:

| Variable | Description |
| --- | --- |
| `STACKIT_AVAILABILITY_ZONE` | STACKIT availability zone |
| `STACKIT_SECURITY_GROUP_ID` | Security group UUID |

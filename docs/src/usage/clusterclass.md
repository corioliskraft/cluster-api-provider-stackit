# ClusterClass

ClusterClass support is provided by:

- `templates/clusterclass.yaml`
- `templates/cluster-template-topology.yaml`

The ClusterClass wires topology variables into STACKIT templates:

- Kubernetes version
- Control-plane replica count
- Worker replica count
- Machine type
- Image ID
- Region
- Network ID
- SSH key name
- Credentials Secret name

Apply the ClusterClass:

```sh
kubectl apply -f templates/clusterclass.yaml
```

Render and apply a topology Cluster:

```sh
clusterctl generate cluster "${CLUSTER_NAME}" \
  --from templates/cluster-template-topology.yaml \
  --kubernetes-version v1.33.12 \
  --control-plane-machine-count 1 \
  --worker-machine-count 1 \
  > "${CLUSTER_NAME}-topology.yaml"

kubectl apply -f "${CLUSTER_NAME}-topology.yaml"
```

The management cluster must run CAPI core and kubeadm-control-plane with
`ClusterTopology=true`. Otherwise the admission webhooks reject `ClusterClass`,
`KubeadmControlPlaneTemplate`, and `Cluster.spec.topology`.

The topology template currently covers the infrastructure, control-plane, and
worker topology wiring only. Unlike the default non-topology
`templates/cluster-template.yaml`, it does not include the
`cloud-provider-stackit` `ClusterResourceSet` addon or the workload-cluster
Secret for the cloud controller manager. Apply equivalent addon wiring before
expecting topology workload Nodes to become Ready with the external cloud
provider.

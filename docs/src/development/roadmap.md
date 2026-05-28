# Roadmap

The project started as an MVP infrastructure provider and is moving through a
contract and reliability hardening phase.

Done or mostly done:

- Kubebuilder provider scaffolding
- STACKIT cluster and machine APIs
- VM lifecycle
- API server load balancer lifecycle
- provider ID compatibility
- clusterctl release packaging
- e2e leak cleanup support
- worker scale coverage
- worker replacement upgrade coverage
- failure domain publication
- ClusterClass templates

Open work:

- full workload-cluster Node readiness with `cloud-provider-stackit`
- full ClusterClass create/ready/delete e2e
- release/distribution decision
- broader production hardening

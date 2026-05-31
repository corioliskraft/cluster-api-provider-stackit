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
- 1 control-plane / 1 worker workload-cluster NodeRef and Node readiness e2e

Open work:

- worker scale Node readiness/removal with `cloud-provider-stackit`
- automatic upgrade rollout and control-plane upgrade readiness with real Nodes
- full ClusterClass create/ready/delete e2e
- release/distribution decision
- broader production hardening

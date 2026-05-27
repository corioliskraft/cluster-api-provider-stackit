# cluster-api-provider-stackit TODO

## Done

- [x] Scaffolded the Kubebuilder provider API types and controllers for `StackitCluster`, `StackitMachine`, `StackitClusterTemplate`, and `StackitMachineTemplate`.
- [x] Added controller reconciliation for existing STACKIT network lookup, API server load balancer creation, VM creation, VM deletion, finalizers, conditions, and status updates.
- [x] Added credential parsing compatible with machine-controller-manager-provider-stackit Secret keys:
  - `project-id`
  - `serviceaccount.json`
- [x] Added a cloud abstraction with classified errors for controller retry decisions.
- [x] Added an in-memory fake cloud client for unit/envtest coverage.
- [x] Added provider ID generation and parsing helpers.
- [x] Verified provider ID format against `cloud-provider-stackit`.
  The current cloud provider generates and accepts `stackit://<server-id>` for STACKIT provider IDs.
- [x] Wired controllers to receive a cloud client factory and configured `cmd/main.go` to use the real client factory.
- [x] Implemented the upstream STACKIT SDK-backed cloud client using:
  - `github.com/stackitcloud/stackit-sdk-go/core v0.26.0`
  - `github.com/stackitcloud/stackit-sdk-go/services/iaas v1.12.0`
  - `github.com/stackitcloud/stackit-sdk-go/services/loadbalancer v1.13.0`
- [x] Implemented SDK-backed server create/get/list/delete, network lookup, load balancer create/list/delete, and load balancer target-pool updates.
- [x] Registered control-plane VM internal IPs as API server load balancer targets.
- [x] Removed control-plane VM targets from the API server load balancer during machine deletion.
- [x] Added focused tests for utility helpers, cloud errors, provider IDs, controller reconciliation, VM lifecycle, load balancer lifecycle, and load balancer target registration/cleanup.
- [x] Added opt-in STACKIT SDK integration tests behind the `stackit_integration` build tag.
  These tests read the local service-account key by default and can validate credentials, load balancer create/delete, and target-pool update/delete against a real project.
- [x] Created a STACKIT `eu01` test network for integration testing:
  - name: `capi-stackit-it-eu01`
  - id: `3a87ac2f-8297-4dea-a9da-11d3c19e45fe`
  - prefix: `10.0.0.0/24`
- [x] Validated the real STACKIT load balancer create/delete and target-pool update/delete payloads against the test network.
- [x] Validated API-server load balancer behavior with a real STACKIT VM target.
  The test VM `capi-stackit-it-target` received internal IP `10.0.0.103` on network `3a87ac2f-8297-4dea-a9da-11d3c19e45fe`; the VM was deleted after validation.
- [x] Confirmed that IAAS NIC `ipv4` is the VM address shape to register as a load balancer target.
- [x] Created the `stackit-credentials` Secret in the local `kind-capi-stackit` cluster.
- [x] Verified the current non-e2e test suite with `make test`.
- [x] Tightened API validation for required provider inputs:
  `credentialsSecretRef.name`, project ID, region, network IDs, image ID, machine type, SSH key names, security group IDs, and the load-balancer/endpoint relationship.
- [x] Updated `config/samples/*` with schema-valid STACKIT sample specs.
- [x] Updated `templates/cluster-template.yaml` for clusterctl rendering, CAPI v1beta2 references, explicit MachineDeployment selectors, and realistic STACKIT disk defaults.
- [x] Added SDK-client unit tests using local HTTP test servers for IAAS server create/NIC translation, load balancer create/update target payloads, and SDK error classification.
- [x] Added controller retry/error-path tests for transient cloud failures, conflicts, unauthorized credentials, missing network, load balancer target update failure, and load balancer deletion failure.
- [x] Fixed controller patching so patch errors are returned, owned conditions are declared, cluster error states mark `status.ready=false`, and empty API endpoints are omitted from status patches.
- [x] Decided `StackitClusterTemplate` and `StackitMachineTemplate` do not need extra MVP fields; they should continue to wrap the concrete provider specs to avoid drift.
- [x] Added related-object watches and mapping tests so:
  - owning `Cluster` updates reconcile the referenced `StackitCluster`
  - owning `Machine` updates reconcile the referenced `StackitMachine`
  - bootstrap Secret updates reconcile dependent `StackitMachine` objects
  - related `StackitCluster` updates reconcile dependent `StackitMachine` objects
- [x] Verified RBAC covers the current controller watch/update paths and regenerated manifests.
- [x] Replaced the scaffold README with STACKIT credentials, template variables, network/security prerequisites, local deployment, testing, and distribution notes.
- [x] Ran an isolated Kind/CAPI e2e smoke test against the real STACKIT project using `kind-capi-stackit`.
  - Installed CAPI core, kubeadm bootstrap, and kubeadm control-plane providers with `clusterctl init`.
  - Built and deployed the local STACKIT infrastructure provider image into kind.
  - Applied a CAPI `Cluster` plus `StackitCluster` for project `4cf9e1f0-1f18-4c5b-bcc5-fbd3dd6675a5`, region `eu01`, and network `3a87ac2f-8297-4dea-a9da-11d3c19e45fe`.
  - Verified `StackitCluster` reached `Ready=True`, `CredentialsReady=True`, and `NetworkReady=True`; CAPI `Cluster` reached `InfrastructureReady=True`.
  - Cleaned up the temporary `stackit-e2e` Cluster/StackitCluster objects.
- [x] Fixed the manager deployment scheme to register Cluster API core v1beta2 types required by CAPI watches.
- [x] Added an opt-in real VM e2e scenario gated by `STACKIT_E2E_CREATE_VMS=true`.
  The scenario applies a CAPI `Cluster`/`Machine` with `StackitCluster`/`StackitMachine`, waits for a real STACKIT server ID, verifies the server with the SDK, deletes the `StackitMachine`, and verifies the server is removed from STACKIT.
- [x] Ran the real VM e2e scenario against the `kind-capi-stackit` cluster and the real STACKIT project.
  The run used non-ARM `Ubuntu 22.04` image `3ad2867e-695b-4ee6-9502-b563013413d4`, machine type `c2i.1`, availability zone `eu01-1`, network `3a87ac2f-8297-4dea-a9da-11d3c19e45fe`, created a real server, verified it through the STACKIT SDK, deleted the `StackitMachine`, and verified no STACKIT server remained.
- [x] Added Cluster API contract status fields `status.initialization.provisioned` for `StackitCluster` and `StackitMachine`.
- [x] Updated API-server load balancer reconciliation to write the provider-managed endpoint to `spec.controlPlaneEndpoint` for the CAPI infrastructure cluster contract.
- [x] Fixed STACKIT SDK create-server boot-volume payloads so explicit root-volume requests use `bootVolume.source` for the image and omit the conflicting top-level `imageId`.
- [x] Added `tutorial.md` with a step-by-step guide for management cluster setup, STACKIT credentials, provider deployment, workload-cluster rendering, real VM e2e validation, cleanup, and troubleshooting.

## Missing

- [x] PR 1: Audit and fix Cluster API infrastructure provider contract behavior.
  - [x] Verify StackitCluster contract fields, status, conditions, finalizer, owner references, paused handling, and observedGeneration.
  - [x] Verify StackitMachine contract fields, `spec.providerID`, status, conditions, finalizer, owner references, paused handling, and observedGeneration.
  - [x] Ensure paused Clusters or paused objects do not trigger cloud API calls.
  - [x] Ensure conditions include observedGeneration and specific reasons.
  - [x] Regenerate CRDs/deepcopy and run unit/envtest coverage.
- [ ] PR 2: Re-verify providerID compatibility with `cloud-provider-stackit`.
  - [x] Confirm providerID format, generation, parsing, and node matching against upstream cloud provider behavior.
  - [x] Keep `NewProviderID`, `ParseProviderID`, round-trip, and cloud-provider-format tests green.
  - [ ] Validate StackitMachine, Machine, and Node providerID alignment in a real workload cluster.
    Static compatibility is verified; real NodeRef validation depends on PR 3 installing `cloud-provider-stackit` in the workload cluster.
- [ ] PR 3: Integrate `cloud-provider-stackit` as an optional workload-cluster addon.
  - [x] Add addon manifests for the STACKIT cloud controller manager and required RBAC/config references.
  - [x] Prepare the workload cluster template for an external cloud provider.
  - [ ] Verify the addon can be applied by e2e tests and nodes become Ready.
- [ ] PR 4: Add reproducible create/delete e2e coverage with leak cleanup.
  - [ ] Create a 1 control-plane / 1 worker cluster.
  - [ ] Delete the cluster and assert no Machines, StackitMachines, VMs, load balancers, or finalizer leaks remain.
  - [ ] Add `make cleanup-stackit` backed by direct STACKIT API cleanup using required test tags.
- [ ] PR 5: Add clusterctl release packaging.
  - [ ] Generate `infrastructure-components.yaml`, `metadata.yaml`, `cluster-template.yaml`, and `cluster-template-development.yaml`.
  - [ ] Verify `clusterctl init --infrastructure stackit` and `clusterctl generate cluster`.
- [ ] PR 6: Add worker scale e2e coverage.
  - [ ] Scale workers from 1 to 3 and back to 1.
  - [ ] Verify providerIDs, node readiness/removal, and no orphaned cloud resources.
- [ ] PR 7: Add Kubernetes upgrade e2e coverage.
  - [ ] Exercise KubeadmControlPlane and MachineDeployment rolling upgrades.
  - [ ] Verify replacement VMs become Ready and old VMs are deleted.
- [ ] PR 8: Model STACKIT availability zones as CAPI FailureDomains.
  - [ ] Publish failureDomains in StackitCluster status.
  - [ ] Validate StackitMachine availability zones and preserve single-AZ templates.
- [ ] PR 9: Add ClusterClass support.
  - [ ] Add ClusterClass and topology templates after create/delete, scale, and upgrade flows are stable.
- [ ] Decide on release/distribution flow: installer YAML, Helm chart, or both.

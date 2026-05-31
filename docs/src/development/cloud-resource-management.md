# Cloud Resource Management Design

This document captures decisions and implementation work for STACKIT resources
that currently sit outside the provider: networks, security groups, SSH keys,
and node images. It is written so a follow-up agent can implement the work in
small, reviewable changes.

## Current State

The provider currently requires users to pass existing cloud resources into the
cluster templates:

- `StackitCluster.spec.network.id` references an existing STACKIT network.
- `StackitMachine.spec.imageID` references an existing image.
- `StackitMachine.spec.sshKeyName` references an optional existing SSH key.
- `StackitMachine.spec.securityGroups` references optional existing security
  group IDs.

`StackitMachineReconciler` reads CABPK/CAPI bootstrap data from
`Machine.spec.bootstrap.dataSecretName` and passes it as STACKIT server user
data during VM creation. There is intentionally no `userData` field on
`StackitMachineSpec`.

The default templates assume the image already has enough OS support for
cloud-init. The real kubeadm e2e fixture currently adds `preKubeadmCommands`
that install and configure `containerd`, `kubelet`, `kubeadm`, and `kubectl` at
runtime from `pkgs.k8s.io`.

## External References

- CAPI kubeadm bootstrap provider: CABPK generates cloud-init data to turn a
  Machine into a Kubernetes Node:
  <https://main.cluster-api.sigs.k8s.io/tasks/bootstrap/kubeadm-bootstrap/>
- Kubernetes kubeadm install guide: kubeadm nodes require a container runtime
  plus `kubeadm`, `kubelet`, and usually `kubectl`:
  <https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/>
- Kubernetes image-builder: repeatable machine images for Cluster API and
  kubeadm-based clusters:
  <https://github.com/kubernetes-sigs/image-builder>
- CAPO configuration: OpenStack keeps image and SSH key as inputs, but supports
  provider-managed security groups:
  <https://cluster-api-openstack.sigs.k8s.io/clusteropenstack/configuration>
- CAPO API reference: `managedSecurityGroups` selects provider-managed security
  groups versus pre-existing groups:
  <https://cluster-api-openstack.sigs.k8s.io/api/v1beta1/api>
- CAPA introduction: AWS provider manages VPCs, gateways, security groups, and
  instances, and does not use SSH for node bootstrap:
  <https://cluster-api-aws.sigs.k8s.io/>
- CAPA CRD reference: SSH key names are optional; security group overrides and
  image lookup/custom AMIs are explicit API fields:
  <https://cluster-api-aws.sigs.k8s.io/crd/>
- STACKIT security groups: virtual firewalls on NICs, with default-deny ingress
  and default-allow egress:
  <https://docs.stackit.cloud/products/network/core-networking/security-groups/basics/concepts/>
- STACKIT IaaS security group API: rules belong to groups; groups are attached
  to NICs; default rules block inbound traffic except same-group traffic:
  <https://docs.stackit.cloud/products/iaas-api/how-tos/manage-security-groups-and-rules-via-iaas-api/>
- STACKIT server creation: server creation accepts image, machine type, network,
  security groups, and SSH key pair:
  <https://docs.stackit.cloud/products/iaas-api/how-tos/create-a-vm-via-iaas-api/>

## Decisions

### Networks

Decision: keep STACKIT network creation outside the provider for now.

Rationale:

- Networks are usually a platform/environment boundary, not a per-cluster
  implementation detail. They encode address planning, routing, DNS, firewall
  posture, peering/VPN, and organization policy.
- Other CAPI providers support both approaches. CAPA can create a VPC for a
  fast path, while CAPO commonly supports existing network/subnet inputs and
  managed subnets. For this provider, the current API and e2e path already rely
  on an existing network and this is reasonable for an MVP and production-safe
  default.
- Creating networks in the provider would introduce lifecycle risk: deleting a
  Cluster could accidentally delete shared infrastructure unless ownership and
  adoption semantics are very explicit.

Implementation stance:

- Do not add automatic network creation to the default `StackitCluster`
  behavior.
- Improve prerequisite docs and examples for creating a network with Terraform
  or STACKIT CLI/API.
- Later, consider an explicit opt-in `network.managed` mode only if there is a
  clear user need. Managed networks must be owned, tagged, non-shared by
  default, and blocked from deletion if adopted/shared.

### Security Groups

Decision: add optional provider-managed security groups, but keep externally
provided security group IDs supported.

Rationale:

- Security groups are directly tied to node correctness and security posture.
  The provider and templates know the minimum traffic needed for kubeadm,
  control-plane API access, `cloud-provider-stackit`, NodePort/load balancer
  behavior, and the chosen CNI.
- STACKIT security groups are NIC-level virtual firewalls. Their default posture
  blocks ingress, which is good, but relying on an unspecified default group
  makes the cluster behavior hard to reason about.
- CAPO is a useful model: it supports `managedSecurityGroups` as an explicit
  opt-in while still allowing pre-existing groups. CAPA manages security groups
  by default but also exposes override fields.

Recommended API direction:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitCluster
spec:
  network:
    id: <network-id>
  managedSecurityGroups:
    enabled: true
    allowAllInClusterTraffic: true
    ssh:
      enabled: false
      allowedCIDRs: []
    apiServer:
      allowedCIDRs:
        - 0.0.0.0/0
    nodePorts:
      enabled: false
      allowedCIDRs: []
```

Recommended defaults:

- Do not enable SSH ingress by default.
- Allow node-to-node traffic within provider-managed control-plane and worker
  groups. This avoids breaking CNI overlays and kubelet/control-plane flows.
- Allow API server ingress according to the load balancer model. Initially this
  can remain broad for parity with current behavior, but the API must allow
  narrowing via CIDRs.
- Keep egress open initially. Add egress restriction only after we document
  exact package, registry, STACKIT API, DNS, and NTP requirements.
- Allow users to pass additional `StackitMachine.spec.securityGroups`; combine
  them with managed groups unless an explicit override mode is added.

Implementation tasks:

1. Extend API types:
   - Add `StackitClusterSpec.ManagedSecurityGroups *StackitManagedSecurityGroupsSpec`.
   - Add status fields for managed security group IDs.
   - Keep `StackitMachineSpec.SecurityGroups []string`.
2. Extend `cloud.Client`:
   - `EnsureSecurityGroup`
   - `EnsureSecurityGroupRule`
   - `DeleteSecurityGroup`
   - `ListSecurityGroupsByTags`
3. Implement SDK support for STACKIT security groups/rules.
4. In `StackitClusterReconciler`, create/update control-plane and worker
   security groups before `StackitCluster.Status.Ready=true`.
5. In `StackitMachineReconciler`, attach role-appropriate managed security
   groups plus explicit machine security groups during server creation.
6. On delete, remove managed security groups after machines/load balancers are
   gone. Treat not-found as success and report conflicts as retryable.
7. Add tests:
   - unit/envtest: desired rules are generated, status IDs set, delete cleans
     up.
   - fake cloud: SG lifecycle and adoption/lookup by tags.
   - e2e: cluster reaches Node Ready with managed security groups and no
     externally supplied group.

Open design details:

- Decide whether control-plane and worker groups should be separate from the
  start. Recommendation: yes, even if initial rules are similar.
- Decide if CNI-specific rules belong in the provider API. Recommendation:
  start with `allowAllInClusterTraffic=true`; add CNI-specific profiles later
  only if users need a stricter policy.

### SSH Keys

Decision: do not create or manage SSH key pairs in the provider. Keep SSH keys
optional and externally managed.

Rationale:

- Cluster API/kubeadm bootstrap does not require SSH. CABPK provides cloud-init
  user data, and CAPA explicitly does not use SSH for bootstrapping nodes.
- SSH keys are identity material, not cluster infrastructure. Creating them in
  a controller raises private-key storage, rotation, audit, and ownership
  questions.
- STACKIT server creation accepts a key pair name. The provider already exposes
  `StackitMachine.spec.sshKeyName`, which matches common provider patterns.

Recommended behavior:

- `sshKeyName` remains optional. Empty means no SSH key is attached.
- Default templates should allow `STACKIT_SSH_KEY_NAME` to be empty.
- Provider-managed security groups must not open port 22 unless an explicit
  `ssh.enabled=true` and CIDR allowlist are configured.
- For production debugging, document safer alternatives first: serial console,
  logs, CAPI conditions/events, cloud-init output, and a bastion/VPN operated
  outside the provider. If SSH is needed, users should create the key and pass
  its name explicitly.

Implementation tasks:

1. Update docs to state SSH is optional and not used for bootstrap.
2. Relax templates so empty `STACKIT_SSH_KEY_NAME` is clearly valid. If
   clusterctl rendering cannot omit empty fields cleanly, add a no-SSH template
   flavor.
3. Add validation/webhook later: if managed SSH ingress is enabled, require
   non-empty allowed CIDRs and a non-empty `sshKeyName`.

### Node Images

Decision: provide a documented default image build pipeline and published image
IDs per Kubernetes minor/region, but do not bake image creation into the
runtime controller.

Rationale:

- Cluster API best practice is to use repeatable kubeadm-ready images, commonly
  built with `kubernetes-sigs/image-builder`.
- Runtime apt installs work for e2e and early development, but they make
  cluster creation slower and less deterministic. They depend on network egress,
  package repository availability, package changes, and apt mirror behavior.
- Images are immutable inputs. The STACKIT server docs note that changing the
  image of an existing server means creating a new server. That maps naturally
  to Machine replacement and rolling upgrades.

What a default image should contain:

- `cloud-init`
- `containerd` configured with systemd cgroups
- `kubelet`
- `kubeadm`
- `kubectl`
- `conntrack`, `iproute2`, `iptables`/`nftables` support, `socat`, `ebtables`
  or distro equivalents required by kubeadm/CNI
- kernel modules/sysctls needed for Kubernetes networking, or boot-time config
  that reliably applies them
- optional debugging basics such as `curl`, `jq`, and `journalctl` availability

Recommendation:

- Build and publish STACKIT images with `image-builder` or an equivalent Packer
  flow.
- Publish a version matrix:
  - Kubernetes minor/patch
  - OS distribution/version
  - STACKIT region
  - image ID
  - architecture
  - build date/source revision
- Keep `STACKIT_IMAGE_ID` as an explicit template input. Do not auto-select an
  image in the controller until there is a robust image discovery API and a
  support matrix.

Implementation tasks:

1. Add `images/` or `hack/image-builder/` with STACKIT Packer/image-builder
   configuration.
2. Add `docs/src/reference/images.md` with the image support matrix.
3. Add `hack/validate-stackit-image.sh` to verify that `STACKIT_IMAGE_ID`
   matches the requested Kubernetes minor and architecture once metadata is
   available.
4. Update templates/e2e:
   - Default user path should assume kubeadm-ready images.
   - Keep runtime install commands only in a development/e2e fallback flavor.
5. Add e2e jobs:
   - kubeadm-ready image path without package-install `preKubeadmCommands`.
   - fallback runtime-install path for development images.

### Runtime Bootstrap Today

Current flow:

1. CABPK/KubeadmConfig generates a bootstrap Secret for each Machine.
2. `StackitMachineReconciler.fetchBootstrapData` reads `value` or `userData`
   from that Secret.
3. `StackitMachineReconciler.ensureServer` passes the bytes to
   `cloud.CreateServerInput.UserData`.
4. `pkg/cloud/sdk_client.go` base64-encodes user data and sends it in the
   STACKIT server create payload.
5. The real NodeRef e2e fixture injects `preKubeadmCommands` that:
   - install `containerd`
   - add the Kubernetes apt repository
   - install `kubelet`, `kubeadm`, and `kubectl`
   - configure containerd systemd cgroups
   - set Kubernetes networking sysctls

This proves the provider works with generic Ubuntu cloud images, but it should
not be the preferred production path.

## Recommended Milestones

### Milestone 1: Documented Prerequisite Flow

Goal: make current behavior production-understandable without changing APIs.

Tasks:

- Add docs for existing network, security group, SSH key, and image inputs.
- Add STACKIT CLI/Terraform examples for creating those resources.
- Clarify that SSH is optional and not used for bootstrap.
- Clarify that the current e2e runtime package install is a development
  fallback.

### Milestone 2: Managed Security Groups

Goal: make default clusters secure and reproducible without requiring users to
manually design firewall rules.

Tasks:

- Add explicit `managedSecurityGroups` API.
- Implement cloud client security group/rule lifecycle.
- Attach managed groups to servers by role.
- Add unit/envtest/fake SDK coverage.
- Add real e2e coverage without user-provided security groups.

### Milestone 3: Kubeadm-Ready Images

Goal: remove package-install variability from the normal workload-cluster path.

Tasks:

- Add image-builder/Packer configuration.
- Publish or document image IDs for supported Kubernetes minors.
- Update docs/templates to prefer kubeadm-ready images.
- Keep runtime install path as `cluster-template-development.yaml` or e2e-only
  fallback.

### Milestone 4: Optional Managed Networks

Goal: decide whether network creation belongs in the provider after the safer
prerequisite and security group flows are stable.

Tasks:

- Gather user demand and concrete network requirements.
- Design `network.managed` with explicit ownership/adoption semantics.
- Ensure deletion cannot remove shared resources accidentally.
- Add e2e coverage in a dedicated STACKIT project/network sandbox.

## Non-Goals For Now

- Provider-generated SSH private keys.
- Automatic mutation of user-owned security groups unless they are explicitly
  marked managed by this provider.
- Automatic image lookup/selection in the controller.
- Default creation/deletion of shared networks.

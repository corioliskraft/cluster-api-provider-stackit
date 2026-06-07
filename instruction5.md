# Bastion Feature Implementation Plan

## Context

This repository is an MVP Cluster API infrastructure provider for STACKIT. It
currently manages:

- `StackitCluster`
- `StackitMachine`
- STACKIT VM creation and deletion
- CAPI bootstrap user data from bootstrap Secrets
- `KubeadmControlPlane` and `MachineDeployment` based workload clusters
- a simple STACKIT VM based workload cluster

The next development phase is to add an optional, provider-managed SSH bastion
host so users can reach cluster nodes from outside the STACKIT network.

The desired user-facing behavior is described in:

```text
docs/src/topics/accessing-vm-instances.md
```

Use the local STACKIT SDK checkout as the main API reference:

```text
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/stackit-sdk-go
```

## Current Code Shape

The provider is currently structured as follows:

- API types live in `api/v1alpha1`.
- `StackitClusterReconciler` in `internal/controller/stackitcluster_controller.go`
  owns cluster-wide infrastructure: credentials, network validation, API server
  load balancer, cluster status, and cluster finalizer cleanup.
- `StackitMachineReconciler` in `internal/controller/stackitmachine_controller.go`
  owns per-machine VM lifecycle and API-server LB target registration.
- Controllers must not use the STACKIT SDK directly. All cloud operations go
  through `pkg/cloud.Client`.
- The SDK-backed implementation is in `pkg/cloud/sdk_client.go`.
- Unit tests use the in-memory fake in `pkg/cloud/fake/client.go`.
- Canonical cloud labels/tags are produced by `pkg/util/tags.go`.

Important existing API details:

- `StackitCluster.spec.network.id` references an existing STACKIT network.
- `StackitMachine.spec.sshKeyName` is optional and maps to the STACKIT server
  `keypairName`.
- `StackitMachine.spec.securityGroups` already supports pre-existing security
  group IDs for node VMs.
- The provider intentionally does not create SSH key pairs.
- `docs/src/topics/cloud-resource-management.md` already states that SSH keys
  stay externally managed and that provider-managed security groups should be
  explicit.

## Goal

Implement an optional `StackitCluster.spec.bastion` feature that creates,
observes, exposes, and deletes one STACKIT bastion VM per workload cluster.

When enabled, the provider must:

- create exactly one bastion server in the cluster network
- attach an existing SSH key pair to that server
- make the bastion reachable from the public Internet by assigning a public IP
- restrict inbound SSH with a STACKIT security group
- publish the bastion public IP in `StackitCluster.status`
- show the bastion IP in `kubectl get stackitclusters`
- delete all provider-managed bastion resources on cluster deletion or when
  bastion is disabled
- keep current default behavior unchanged when bastion is not enabled

## Non-goals

Do not implement these in this phase:

- STACKIT network creation
- SSH key pair creation or private key storage
- direct public IPs on control-plane or worker nodes
- SKE support
- high availability or multiple bastions
- VPN support
- Windows node support
- CNI-specific strict firewall profiles
- production-grade egress lockdown

## API Design

Add a cluster-level bastion spec to `api/v1alpha1/stackitcluster_types.go`:

```go
type StackitBastionSpec struct {
    // enabled toggles creation of a provider-managed SSH bastion.
    // +optional
    Enabled bool `json:"enabled,omitempty"`

    // imageID is the STACKIT image used for the bastion VM.
    // Required when enabled is true.
    // +optional
    // +kubebuilder:validation:Pattern=`^[0-9a-fA-F-]{36}$`
    ImageID string `json:"imageID,omitempty"`

    // machineType is the STACKIT machine type used for the bastion VM.
    // Required when enabled is true unless a default is added.
    // +optional
    // +kubebuilder:validation:Pattern=`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`
    MachineType string `json:"machineType,omitempty"`

    // sshKeyName is the name of an existing STACKIT SSH key pair.
    // Required when enabled is true.
    // +optional
    // +kubebuilder:validation:MaxLength=127
    // +kubebuilder:validation:Pattern=`^[A-Za-z0-9_.@-]+$`
    SSHKeyName string `json:"sshKeyName,omitempty"`

    // allowedCIDRs are CIDR ranges allowed to SSH to the bastion.
    // Required when enabled is true. Use ["0.0.0.0/0"] only for the explicit
    // broad-access behavior documented today.
    // +optional
    // +kubebuilder:validation:items:Format=cidr
    AllowedCIDRs []string `json:"allowedCIDRs,omitempty"`

    // rootVolume optionally configures the bastion root disk.
    // +optional
    RootVolume StackitRootVolumeSpec `json:"rootVolume,omitempty"`
}
```

Add this field to `StackitClusterSpec`:

```go
// bastion configures an optional provider-managed SSH bastion.
// +optional
Bastion StackitBastionSpec `json:"bastion,omitempty"`
```

Validation:

- Prefer CRD CEL validation if controller-tools supports it cleanly:
  `!self.bastion.enabled || (...)`.
- At minimum, controller validation must reject or condition-fail enabled
  bastion specs with empty `imageID`, `machineType`, `sshKeyName`, or
  `allowedCIDRs`.
- `allowedCIDRs` must contain valid CIDRs and must not contain empty strings.
- Keep bastion disabled by default.

Do not use the `ami` field from the current draft doc. STACKIT uses image IDs,
so the public API field must be `imageID`.

## Status Design

Add a nested bastion status to `StackitClusterStatus`:

```go
type StackitBastionStatus struct {
    // serverID is the provider-managed bastion server ID.
    // +optional
    ServerID string `json:"serverID,omitempty"`

    // publicIPID is the provider-managed public IP resource ID.
    // +optional
    PublicIPID string `json:"publicIPID,omitempty"`

    // publicIP is the routable IP address users connect to.
    // +optional
    PublicIP string `json:"publicIP,omitempty"`

    // securityGroupID is the provider-managed bastion security group ID.
    // +optional
    SecurityGroupID string `json:"securityGroupID,omitempty"`
}
```

Add to `StackitClusterStatus`:

```go
// bastion stores observed provider-managed bastion resources.
// +optional
Bastion StackitBastionStatus `json:"bastion,omitempty"`
```

Add condition type:

```go
ClusterBastionReadyCondition = "BastionReady"
```

Condition behavior:

- disabled: `BastionReady=True`, reason `Skipped`, message `bastion disabled`
- provisioning: `BastionReady=False`, reason `Provisioning`
- invalid spec: `BastionReady=False`, reason `InvalidBastionSpec`
- cloud error: `BastionReady=False`, reason `BastionError`
- ready: `BastionReady=True`, reason `Available`

Add a StackitCluster printcolumn for `.status.bastion.publicIP` named
`Bastion IP`.

## Cloud Abstraction

Extend `pkg/cloud/types.go` and `pkg/cloud/client.go`. Keep controllers free of
SDK structs.

Recommended provider-neutral types:

```go
type BastionInput struct {
    Name         string
    ProjectID    string
    Region       string
    NetworkID    string
    ImageID      string
    MachineType  string
    SSHKeyName   string
    AllowedCIDRs []string
    Tags         map[string]string
    RootVolume   RootVolumeInput
}

type Bastion struct {
    ServerID        string
    ServerState     string
    PublicIPID      string
    PublicIP        string
    SecurityGroupID string
}
```

Recommended client methods:

```go
EnsureBastion(ctx context.Context, input BastionInput) (*Bastion, error)
DeleteBastion(ctx context.Context, input BastionInput, status Bastion) error
ListPublicIPsByTags(ctx context.Context, tags map[string]string) ([]*PublicIP, error)
ListSecurityGroupsByTags(ctx context.Context, tags map[string]string) ([]*SecurityGroup, error)
```

It is also acceptable to expose smaller primitives instead of
`EnsureBastion`, but keep the controller simple and keep idempotency inside
`pkg/cloud`.

Implementation rules:

- All create/ensure operations must be idempotent by tag lookup.
- Use `util.ClusterTags(...)` plus one extra reserved tag identifying the
  bastion resource role, for example
  `cluster-api-provider-stackit/resource-role=bastion`.
- Do not let user-supplied labels overwrite provider-reserved labels.
- Treat not-found during delete as success.
- Treat attach/detach conflicts as retryable unless the SDK response clearly
  means the desired state already exists.

## STACKIT SDK Work

Use the local SDK checkout to implement the SDK-backed methods. Relevant v2
IAAS API methods exist in:

```text
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/stackit-sdk-go/services/iaas/v2api
```

Known relevant methods:

```text
CreatePublicIP(ctx, projectID, region)
GetPublicIP(ctx, projectID, region, publicIPID)
ListPublicIPs(ctx, projectID, region)
UpdatePublicIP(ctx, projectID, region, publicIPID)
DeletePublicIP(ctx, projectID, region, publicIPID)
AddPublicIpToServer(ctx, projectID, region, serverID, publicIPID)
RemovePublicIpFromServer(ctx, projectID, region, serverID, publicIPID)

CreateSecurityGroup(ctx, projectID, region)
ListSecurityGroups(ctx, projectID, region)
DeleteSecurityGroup(ctx, projectID, region, securityGroupID)
CreateSecurityGroupRule(ctx, projectID, region, securityGroupID)
ListSecurityGroupRules(ctx, projectID, region, securityGroupID)
DeleteSecurityGroupRule(ctx, projectID, region, securityGroupID, ruleID)
AddSecurityGroupToServer(ctx, projectID, region, serverID, securityGroupID)
RemoveSecurityGroupFromServer(ctx, projectID, region, serverID, securityGroupID)

ListServerNICs(ctx, projectID, region, serverID)
```

Useful SDK model details:

- `CreatePublicIPPayload` supports `labels` and `networkInterface`.
- `PublicIp` has `id`, `ip`, `labels`, and `networkInterface`.
- `CreateSecurityGroupPayload` requires `name`, defaults `stateful=true`, and
  supports `description`, `labels`, and initial `rules`.
- `CreateSecurityGroupRulePayload` requires `direction` and supports
  `ethertype`, `ipRange`, `portRange`, `remoteSecurityGroupId`, and `protocol`.

Implementation guidance:

1. Create or find the security group by tags.
2. Ensure one ingress TCP/22 rule per `allowedCIDRs` entry.
3. Keep egress open for the MVP unless STACKIT default egress is confirmed to
   be sufficient. If explicit egress rules are needed, add them to the managed
   security group.
4. Create or find the bastion server by tags using existing `CreateServer`.
5. Attach the managed security group to the bastion server.
6. Create or find the public IP by tags.
7. Attach the public IP to the bastion server. If direct server attach does
   not yield stable results, list the bastion NICs and use public IP
   `networkInterface` association instead.
8. Return `PublicIP` only when the public IP object has a non-empty `ip`.

## Controller Workflow

Add bastion reconciliation to `StackitClusterReconciler`.

Normal reconciliation order:

1. Add finalizer.
2. Publish failure domains.
3. Build cloud client from credentials.
4. Validate the configured network exists.
5. Reconcile API server load balancer as today.
6. Reconcile bastion:
   - if disabled, delete any previously managed bastion resources recorded in
     status, clear bastion status, set `BastionReady=True/Skipped`
   - if enabled, validate bastion spec
   - call `cloudClient.EnsureBastion`
   - store `serverID`, `publicIPID`, `publicIP`, and `securityGroupID`
   - if server is not `ACTIVE` or public IP is empty, set provisioning
     condition and requeue after a short interval
   - if ready, set `BastionReady=True/Available`
7. Set cluster ready only when network, API endpoint, and enabled bastion are
   ready.

Delete reconciliation:

1. Build the cloud client.
2. Delete API server load balancer as today.
3. Delete provider-managed bastion resources.
4. Clear bastion status.
5. Remove finalizer.

Deletion order for bastion resources:

1. Remove/detach public IP from server when needed.
2. Delete bastion server.
3. Delete public IP.
4. Remove security group from server if still attached.
5. Delete security group rules if needed.
6. Delete security group.

Be tolerant of asynchronous cloud deletion:

- Return `ctrl.Result{RequeueAfter: ...}` for transient deletion conflicts.
- Do not remove the finalizer until managed bastion resources are gone or
  confirmed not found.

## SSH Semantics

The bastion only provides network reachability. It does not inject SSH access
into node VMs.

Users must configure:

- `StackitCluster.spec.bastion.sshKeyName` for the bastion server
- `StackitMachine.spec.sshKeyName` on control-plane and worker machine
  templates if they want to SSH into nodes through the bastion

Keep node `sshKeyName` optional. Clusters without SSH keys must continue to
work for Kubernetes bootstrap.

Documentation must clearly say that leaving node `sshKeyName` empty means
ProxyCommand SSH to nodes will not work, even if the bastion is reachable.

## Templates and clusterctl Variables

Update `templates/cluster-template.yaml`,
`templates/cluster-template-development.yaml`, and topology/ClusterClass
templates only after the API and controller are implemented.

Add variables to `docs/src/reference/variables.md`:

```text
STACKIT_BASTION_ENABLED
STACKIT_BASTION_IMAGE_ID
STACKIT_BASTION_MACHINE_TYPE
STACKIT_BASTION_SSH_KEY_NAME
STACKIT_BASTION_ALLOWED_CIDRS
```

Clusterctl templates do not have rich conditionals. Prefer one of these
approaches:

1. Keep the default template bastion-disabled and document a manifest patch for
   enabling bastion.
2. Add a separate bastion template flavor if unconditional empty fields make
   validation awkward.
3. Use `enabled: ${STACKIT_BASTION_ENABLED}` and ensure the controller only
   validates other bastion fields when enabled is true.

Do not reintroduce the old `ami` field in templates or docs.

## Documentation

Update `docs/src/topics/accessing-vm-instances.md` after implementation.

Required doc corrections:

- Replace `ami` with `imageID`.
- Do not call STACKIT networking a VPC or assume AWS/EKS terminology.
- Explain that the bastion uses a STACKIT public IP, not necessarily a
  separate public subnet.
- Explain that SSH keys are externally managed and not used for bootstrap.
- Show that both bastion and nodes need a STACKIT SSH key name for SSH access.
- Show how to restrict `allowedCIDRs`.
- Show how to read the bastion IP from:
  `kubectl get stackitcluster <name> -o jsonpath='{.status.bastion.publicIP}'`
- Remove the EKS-specific command.
- Replace placeholder STACKIT CLI commands with real commands or remove that
  section until it is verified.

Also update:

- `docs/src/topics/cloud-resource-management.md`
- `docs/src/topics/iam-permissions.md`
- `docs/src/reference/variables.md`
- `hack/tf/iam-setup` role permissions

## IAM Permissions

The strict provider role currently covers VM, network, server NIC, and NLB
operations. Bastion support needs additional permissions. Verify exact STACKIT
permission names empirically before finalizing docs and OpenTofu.

Expected new permission families:

```text
iaas.public-ip.create
iaas.public-ip.delete
iaas.public-ip.get
iaas.public-ip.list
iaas.public-ip.update
iaas.server.public-ip.add
iaas.server.public-ip.remove
iaas.security-group.create
iaas.security-group.delete
iaas.security-group.get
iaas.security-group.list
iaas.security-group.rule.create
iaas.security-group.rule.delete
iaas.security-group.rule.list
iaas.server.security-group.add
iaas.server.security-group.remove
```

Do not trust this list blindly. Use the SDK call list, STACKIT IAM docs, and a
strict-role e2e run to prove the final set.

Update `hack/tf/iam-setup` so the generated
`cluster-api-provider-stackit` role includes the final verified permission
set.

## Tests

Add focused tests before or with each implementation layer.

Cloud fake:

- Track bastion servers, public IPs, security groups, rules, and attachments.
- Provide failure injection for public IP, security group, and attach/detach
  operations.
- Add helper counters for leak assertions.

SDK client unit tests:

- Public IP payload contains labels.
- Security group payload contains name, labels, and stateful behavior.
- SSH rule payload uses ingress, IPv4, TCP, port 22, and expected CIDR.
- Idempotent lookup by labels works for public IPs and security groups.
- Not-found, unauthorized, invalid-input, conflict, and transient errors are
  classified consistently with existing SDK code.

StackitCluster controller tests:

- Bastion disabled keeps current behavior and creates no bastion resources.
- Enabled bastion with valid spec creates server, public IP, security group,
  rule, and status.
- Enabled bastion sets `Ready=False` while server is not `ACTIVE`.
- Enabled bastion sets `Ready=False` while public IP is not assigned yet.
- Invalid enabled spec sets `BastionReady=False/InvalidBastionSpec` and does
  not call cloud create methods.
- Reconcile is idempotent and does not duplicate bastion resources.
- Lost status can be recovered by tag lookup.
- Toggling `spec.bastion.enabled` from true to false deletes managed bastion
  resources and clears bastion status.
- Cluster deletion deletes API LB and bastion resources before removing the
  finalizer.
- Delete tolerates not-found resources.
- Delete requeues on transient cloud conflicts.

End-to-end/manual validation:

1. Create/update the strict IAM role with bastion permissions.
2. Create a workload cluster with bastion enabled.
3. Configure `StackitMachineTemplate.spec.template.spec.sshKeyName` for both
   control-plane and worker nodes.
4. Wait for `StackitCluster` and all `StackitMachine` objects to be ready.
5. Confirm `.status.bastion.publicIP` is non-empty.
6. Confirm TCP/22 is reachable on the bastion from an allowed CIDR.
7. SSH through the bastion to a control-plane node and worker node by internal
   IP using `ProxyCommand`.
8. Delete the cluster.
9. Use a broader inspection service account to verify no provider-managed
   bastion server, public IP, security group, security group rule, or API LB
   remains.

Run at least:

```sh
make generate
make manifests
make test
```

For cloud validation, run the existing e2e create/delete path plus a new
bastion-specific scenario.

## Acceptance Criteria

The feature is complete when:

- Bastion is disabled by default.
- Existing cluster templates still work without SSH or bastion configuration.
- Enabling `StackitCluster.spec.bastion.enabled=true` creates one tagged
  bastion server, one tagged public IP, and one tagged bastion security group.
- The bastion has inbound TCP/22 open only from configured `allowedCIDRs`.
- `StackitCluster.status.bastion.publicIP` is populated after the public IP is
  assigned.
- `kubectl get stackitclusters` shows a `Bastion IP` column.
- Users can SSH to node internal IPs through the bastion when node machine
  templates also set `sshKeyName`.
- Cluster deletion or bastion disablement cleans up all provider-managed
  bastion resources.
- The least-privilege OpenTofu role and IAM docs include the verified new
  permissions.
- Unit tests and relevant e2e tests pass.

Use this admin service account for inspection:
.stackit/serviceaccount.json

Make sure the opentofu generated service account is used for e2e tests.

## Recommended PR Order

Implement in small, reviewable changes:

```text
PR 1: SDK spike and cloud abstraction for public IPs/security groups
PR 2: API/status fields for StackitCluster bastion plus generated manifests
PR 3: fake cloud support and cloud-level unit tests
PR 4: StackitCluster bastion reconciliation and deletion
PR 5: controller tests for create/idempotency/disable/delete/error paths
PR 6: templates, docs, IAM OpenTofu update
PR 7: bastion e2e/manual validation and final permission proof
```

If using Jujutsu, use separate bookmarks or clearly described changes for
these steps. Do not bundle the full feature into one large change.

## Open Questions to Resolve During PR 1

- Does `AddPublicIpToServer` reliably attach the public IP to the intended
  bastion NIC, or should the implementation explicitly list NICs and associate
  the public IP with `networkInterface`?
- Are security groups attached to servers or NICs in the relevant STACKIT API
  path, and does `AddSecurityGroupToServer` cover the desired behavior?
- What exact permission strings does STACKIT IAM require for public IP,
  security group, rule, and attach/detach calls?
- What machine type should be documented as the default bastion size?
- Which Ubuntu image ID should examples use per region, and should docs avoid
  hard-coding one region-specific image ID?
- Does the current node image have SSH enabled for the expected user `ubuntu`?
  If not, document the correct user or image prerequisite.

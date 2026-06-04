resource "stackit_authorization_project_custom_role" "cluster_api_provider_stackit" {
  resource_id = var.project_id
  name        = var.role_name
  description = "Least-privilege role for cluster-api-provider-stackit VM, bastion, and API-server load-balancer reconciliation."

  permissions = [
    "iaas.network.get",
    "iaas.public-ip.create",
    "iaas.public-ip.delete",
    "iaas.public-ip.get",
    "iaas.public-ip.list",
    "iaas.server.create",
    "iaas.server.delete",
    "iaas.server.get",
    "iaas.server.list",
    "iaas.server.nic.list",
    "iaas.server.public-ip.add",
    "iaas.server.public-ip.remove",
    "iaas.server.security-group.add",
    "iaas.server.security-group.remove",
    "iaas.security-group.create",
    "iaas.security-group.delete",
    "iaas.security-group.list",
    "iaas.security-group.get",
    "iaas.security-group.rule.create",
    "iaas.security-group.rule.delete",
    "iaas.security-group.rule.list",
    "iaas.security-group.rule.get",
    "nlb.loadbalancer.create",
    "nlb.loadbalancer.delete",
    "nlb.loadbalancer.get",
    "nlb.loadbalancer.list",
    "nlb.targetpool.replace",
  ]
}

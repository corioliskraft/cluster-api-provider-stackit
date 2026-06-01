resource "stackit_authorization_project_custom_role" "cluster_api_provider_stackit" {
  resource_id = var.project_id
  name        = var.role_name
  description = "Least-privilege role for cluster-api-provider-stackit VM and API-server load-balancer reconciliation."

  permissions = [
    "iaas.network.get",
    "iaas.server.create",
    "iaas.server.delete",
    "iaas.server.get",
    "iaas.server.list",
    "iaas.server.nic.list",
    "nlb.loadbalancer.create",
    "nlb.loadbalancer.delete",
    "nlb.loadbalancer.get",
    "nlb.loadbalancer.list",
    "nlb.targetpool.replace",
  ]
}

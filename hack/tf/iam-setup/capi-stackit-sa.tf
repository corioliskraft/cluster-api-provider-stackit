resource "stackit_service_account" "cluster_api_provider_stackit" {
  project_id = var.project_id
  name       = var.service_account_name
}

resource "stackit_authorization_project_role_assignment" "cluster_api_provider_stackit" {
  resource_id = var.project_id
  role        = stackit_authorization_project_custom_role.cluster_api_provider_stackit.name
  subject     = stackit_service_account.cluster_api_provider_stackit.email
}

resource "stackit_service_account_key" "cluster_api_provider_stackit" {
  project_id            = var.project_id
  service_account_email = stackit_service_account.cluster_api_provider_stackit.email

  depends_on = [
    stackit_authorization_project_role_assignment.cluster_api_provider_stackit,
  ]
}

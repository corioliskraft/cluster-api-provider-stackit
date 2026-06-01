output "role_id" {
  description = "ID of the strict custom role."
  value       = stackit_authorization_project_custom_role.cluster_api_provider_stackit.role_id
}

output "project_id" {
  description = "STACKIT project ID."
  value       = var.project_id
}

output "role_name" {
  description = "Name of the strict custom role."
  value       = stackit_authorization_project_custom_role.cluster_api_provider_stackit.name
}

output "service_account_email" {
  description = "Email of the service account used by cluster-api-provider-stackit."
  value       = stackit_service_account.cluster_api_provider_stackit.email
}

output "service_account_id" {
  description = "Internal UUID of the service account."
  value       = stackit_service_account.cluster_api_provider_stackit.service_account_id
}

output "service_account_key_id" {
  description = "ID of the generated service-account key."
  value       = stackit_service_account_key.cluster_api_provider_stackit.key_id
}

output "service_account_key_json" {
  description = "Generated service-account key JSON. Write this to the Kubernetes Secret input file."
  value       = stackit_service_account_key.cluster_api_provider_stackit.json
  sensitive   = true
}

variable "project_id" {
  description = "STACKIT project ID where the provider will manage infrastructure."
  type        = string
}

variable "region" {
  description = "Default STACKIT region for provider configuration."
  type        = string
  default     = "eu01"
}

variable "bootstrap_service_account_key_path" {
  description = "Path to a STACKIT service-account key with IAM setup permissions."
  type        = string
  sensitive   = true
}

variable "role_name" {
  description = "Name of the strict custom role to create."
  type        = string
  default     = "cluster-api-provider-stackit"
}

variable "service_account_name" {
  description = "Prefix for the provider service account. STACKIT requires at most 20 characters."
  type        = string
  default     = "capi-stackit"

  validation {
    condition     = length(var.service_account_name) <= 20
    error_message = "STACKIT service-account name prefixes must be at most 20 characters."
  }
}

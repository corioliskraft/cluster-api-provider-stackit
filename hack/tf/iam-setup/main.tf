terraform {
  required_providers {
    stackit = {
      source  = "stackitcloud/stackit"
      version = "~> 0.98"
    }
  }
}

provider "stackit" {
  default_region           = var.region
  service_account_key_path = var.bootstrap_service_account_key_path
  experiments              = ["iam"]
}

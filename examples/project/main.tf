terraform {
  required_providers {
    evroc = {
      source = "github.com/evroc-oss/evroc"
    }
  }
}

provider "evroc" {
  # Credentials and context read from ~/.evroc/config.yaml by default.
  # See: https://github.com/evroc-oss/terraform-provider-evroc#1-authenticate-with-evroc
}

# Create a new project within your organization
resource "evroc_project" "dev" {
  name         = "dev-environment"
  display_name = "Development Environment"
  organization = "your-organization-id" # Replace with your organization ID

  user_labels = {
    environment = "development"
    team        = "platform"
    cost-center = "engineering"
  }
}

# Create another project for staging
resource "evroc_project" "staging" {
  name         = "staging-environment"
  display_name = "Staging Environment"
  organization = "your-organization-id" # Replace with your organization ID

  user_labels = {
    environment = "staging"
    team        = "platform"
    cost-center = "engineering"
  }
}

# Check organization quota usage
data "evroc_organization_quota" "current" {}

# Check project quota usage
data "evroc_project_quota" "current" {}

# Output the project IDs
output "dev_project_id" {
  description = "UUID of the dev project"
  value       = evroc_project.dev.project_id
}

output "dev_project_name" {
  description = "Name of the dev project"
  value       = evroc_project.dev.name
}

output "staging_project_id" {
  description = "UUID of the staging project"
  value       = evroc_project.staging.project_id
}

output "org_compute_vcpus" {
  description = "Organization vCPU quota"
  value       = "${data.evroc_organization_quota.current.usage_vcpus} / ${data.evroc_organization_quota.current.compute_vcpus} vCPUs"
}

output "org_public_ips" {
  description = "Organization public IP quota"
  value       = "${data.evroc_organization_quota.current.usage_public_ips} / ${data.evroc_organization_quota.current.networking_public_ips} public IPs"
}

output "project_storage" {
  description = "Project object storage quota"
  value       = "limit: ${data.evroc_project_quota.current.object_storage_total_size}"
}

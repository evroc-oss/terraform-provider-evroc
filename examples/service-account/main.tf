terraform {
  required_providers {
    evroc = {
      source = "github.com/evroc-oss/evroc"
    }
  }
}

provider "evroc" {
  # Credentials and context read from ~/.evroc/config.yaml by default.
  #
  # For CI/CD pipelines, authenticate with a service account instead:
  #
  #   service_account_id     = var.evroc_sa_id
  #   service_account_secret = var.evroc_sa_secret
  #   project                = "your-project-id"
}

# ---------------------------------------------------------------------------
# Step 1: Discover available roles
# ---------------------------------------------------------------------------
# Use this data source to see what roles exist before assigning them.
# Run `terraform plan` and check the output to browse the full catalog.

data "evroc_roles" "all" {}

output "available_roles" {
  description = "All available IAM roles"
  value       = [for r in data.evroc_roles.all.roles : "${r.id} — ${r.description} (${r.scope})"]
}

# ---------------------------------------------------------------------------
# Step 2: Create service accounts
# ---------------------------------------------------------------------------

resource "evroc_service_account" "cicd" {
  name        = "sa-cicd-pipeline"
  description = "Service account for CI/CD pipeline automation"
  enabled     = true
}

resource "evroc_service_account" "monitoring" {
  name        = "sa-monitoring"
  description = "Read-only service account for monitoring and alerting"
  enabled     = true
}

# ---------------------------------------------------------------------------
# Step 3: Assign roles — without roles a service account cannot do anything
# ---------------------------------------------------------------------------

# CI/CD: full infrastructure management (compute, disks, networking, storage)
resource "evroc_role_binding" "cicd_infra" {
  principal = evroc_service_account.cicd.fqid
  role      = "cicdInfrastructureManager"
}

resource "evroc_role_binding" "cicd_lb" {
  principal = evroc_service_account.cicd.fqid
  role      = "loadBalancerOperator"
}

# Monitoring: read-only access to observe resources
resource "evroc_role_binding" "monitoring_compute" {
  principal = evroc_service_account.monitoring.fqid
  role      = "computeViewer"
}

resource "evroc_role_binding" "monitoring_networking" {
  principal = evroc_service_account.monitoring.fqid
  role      = "networkingViewer"
}

# ---------------------------------------------------------------------------
# Step 4: Create credentials — the private key is only returned once
# ---------------------------------------------------------------------------

resource "evroc_service_account_credential" "cicd_key" {
  name                  = "cicd-key-2026"
  service_account_ref   = evroc_service_account.cicd.fqid
  description           = "CI/CD pipeline credential — rotated annually"
  expires_at            = "2027-01-01T00:00:00Z"
  access_token_lifetime = 3600
}

resource "evroc_service_account_credential" "monitoring_key" {
  name                  = "monitoring-key-2026"
  service_account_ref   = evroc_service_account.monitoring.fqid
  description           = "Monitoring credential"
  expires_at            = "2027-01-01T00:00:00Z"
  access_token_lifetime = 3600
}

# ---------------------------------------------------------------------------
# Outputs — use these to configure your CI/CD runner
# ---------------------------------------------------------------------------

output "cicd_private_key_jwk" {
  description = "Private key for the CI/CD service account (store securely, shown only once)"
  value       = evroc_service_account_credential.cicd_key.private_key_jwk
  sensitive   = true
}

output "monitoring_private_key_jwk" {
  description = "Private key for the monitoring service account"
  value       = evroc_service_account_credential.monitoring_key.private_key_jwk
  sensitive   = true
}

# ---------------------------------------------------------------------------
# After applying, configure your CI/CD runner:
#
#   export EVROC_SERVICE_ACCOUNT_ID=sa-cicd-pipeline
#   export EVROC_SERVICE_ACCOUNT_SECRET=$(terraform output -raw cicd_private_key_jwk)
#   export EVROC_PROJECT=<your-project>
#   export EVROC_REGION=se-sto
#
# The CI/CD service account can now create VMs, disks, public IPs, security
# groups, buckets, and load balancers — but cannot manage IAM or other users.
#
# The monitoring service account can list and inspect compute and networking
# resources, but cannot create or modify them.
# ---------------------------------------------------------------------------

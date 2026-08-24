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

# Create a simple bucket
resource "evroc_bucket" "app_data" {
  name                  = "app-data-bucket"
  object_retention_mode = "Disabled"
}

# Create a bucket with versioning and lifecycle rules
resource "evroc_bucket" "versioned_data" {
  name                  = "versioned-data-bucket"
  object_retention_mode = "Versioned"

  # Expire objects under tmp/ after 30 days
  lifecycle_rule {
    id = "expire-tmp"

    expire_current_version {
      days = 30
    }

    filter {
      prefix = "tmp/"
    }
  }

  # Keep at most 5 old versions, drop versions older than 90 days,
  # and abort incomplete multipart uploads after 7 days
  lifecycle_rule {
    id = "cleanup-versions"

    expire_non_current_version {
      days             = 90
      max_num_versions = 5
    }

    abort_incomplete_multipart {
      days = 7
    }
  }
}

# Create a bucket with object locking
resource "evroc_bucket" "locked_data" {
  name                  = "locked-data-bucket"
  object_retention_mode = "Locking"

  object_locking {
    mode          = "Soft"
    duration_days = 30
  }
}

# Create a service account for app access
resource "evroc_bucket_service_account" "app_access" {
  name = "app-service-account"

  buckets = [
    evroc_bucket.app_data.name,
    evroc_bucket.versioned_data.name
  ]
}

# Create a service account for backups
resource "evroc_bucket_service_account" "backup_access" {
  name = "backup-service-account"

  buckets = [
    evroc_bucket.locked_data.name
  ]
}

# Query bucket information
data "evroc_bucket" "app_data" {
  name = evroc_bucket.app_data.name
}

# Query service account information
data "evroc_bucket_service_account" "app_access" {
  name = evroc_bucket_service_account.app_access.name
}

data "evroc_bucket_service_account_secret" "app_access" {
  name = evroc_bucket_service_account.app_access.credentials_secret
}

# Output bucket details
output "app_data_bucket_id" {
  value       = evroc_bucket.app_data.bucket_id
  description = "ID of the app data bucket"
}

output "locked_bucket_retention" {
  value       = evroc_bucket.locked_data.object_retention_mode
  description = "Retention mode of the locked bucket"
}

# Output service account details
output "app_service_account_id" {
  value       = evroc_bucket_service_account.app_access.service_account_id
  description = "ID of the app service account"
}

output "app_credentials_secret" {
  value       = evroc_bucket_service_account.app_access.credentials_secret
  description = "Kubernetes secret containing S3 credentials for app access"
  sensitive   = true
}

output "backup_credentials_secret" {
  value       = evroc_bucket_service_account.backup_access.credentials_secret
  description = "Kubernetes secret containing S3 credentials for backup access"
  sensitive   = true
}

output "accessible_buckets" {
  value       = data.evroc_bucket_service_account.app_access.buckets
  description = "List of buckets accessible by app service account"
}

# S3 credentials for direct use (e.g., in app config)
output "s3_access_key_id" {
  value       = data.evroc_bucket_service_account_secret.app_access.access_key_id
  description = "S3 access key ID for the app service account"
  sensitive   = true
}

output "s3_secret_access_key" {
  value       = data.evroc_bucket_service_account_secret.app_access.secret_access_key
  description = "S3 secret access key for the app service account"
  sensitive   = true
}

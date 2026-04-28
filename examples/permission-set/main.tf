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

# Grant a user read-only access to a project
resource "evroc_permission_set" "developer" {
  name    = "dev-alice"
  project = "your-project-id" # Replace with your project ID
  email   = "alice@example.com"
  admin   = false

  user_labels = {
    role       = "developer"
    department = "engineering"
  }
}

# Grant a user admin access to a project
resource "evroc_permission_set" "admin" {
  name    = "admin-bob"
  project = "your-project-id" # Replace with your project ID
  email   = "bob@example.com"
  admin   = true

  user_labels = {
    role       = "administrator"
    department = "platform"
  }
}

# Look up an existing permission set
data "evroc_permission_set" "existing" {
  name = evroc_permission_set.developer.name
}

# Outputs
output "developer_permission_set_id" {
  description = "UUID of the developer permission set"
  value       = evroc_permission_set.developer.permission_set_id
}

output "admin_permission_set_id" {
  description = "UUID of the admin permission set"
  value       = evroc_permission_set.admin.permission_set_id
}

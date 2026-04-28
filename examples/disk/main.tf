terraform {
  required_providers {
    evroc = {
      source  = "github.com/evroc-oss/evroc"
      version = "~> 0.1"
    }
  }
}

provider "evroc" {
  # Credentials and context read from ~/.evroc/config.yaml by default.
  # See: https://github.com/evroc-oss/terraform-provider-evroc#1-authenticate-with-evroc
}

# Create a persistent disk
resource "evroc_disk" "example" {
  name  = "example-disk"
  size  = 100
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

# Output the disk details
output "disk_id" {
  value       = evroc_disk.example.disk_id
  description = "The unique ID of the disk"
}

output "disk_created_at" {
  value       = evroc_disk.example.created_at
  description = "When the disk was created"
}

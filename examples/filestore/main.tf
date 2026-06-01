terraform {
  required_providers {
    evroc = {
      source = "registry.terraform.io/evroc/evroc"
    }
  }
}

provider "evroc" {
  # Credentials and context read from ~/.evroc/config.yaml by default.
}

# Create a shared NFS file store
resource "evroc_filestore" "shared" {
  name = "team-shared-fs"
  zone = "a"

  user_labels = {
    team = "platform"
    env  = "production"
  }
}

# Look up an existing file store
data "evroc_filestore" "shared" {
  name = evroc_filestore.shared.name
}

# Output NFS mount details
output "nfs_endpoint" {
  value       = evroc_filestore.shared.nfs_endpoint
  description = "NFS server address"
}

output "nfs_export_path" {
  value       = evroc_filestore.shared.nfs_export_path
  description = "NFS export path"
}

output "mount_command" {
  value       = "sudo mkdir -p /mnt/shared && sudo mount -t nfs4 -o vers=4.1 ${evroc_filestore.shared.nfs_endpoint}:${evroc_filestore.shared.nfs_export_path} /mnt/shared"
  description = "Command to mount the NFS share"
}

output "status" {
  value = evroc_filestore.shared.status
}

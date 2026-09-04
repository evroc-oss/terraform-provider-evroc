terraform {
  required_providers {
    evroc = {
      source = "github.com/evroc-oss/evroc"
    }
  }
}

# The provider needs an existing project for authentication.
# This is the "bootstrap" project — you must have access to at least one project.
provider "evroc" {}

variable "admin_user_id" {
  description = "UUID of the user to grant owner access on the new project (find yours with the evroc CLI or console)"
  type        = string
}

# --- Step 1: Create a new project ---
# Organization defaults to the one in ~/.evroc/config.yaml.

resource "evroc_project" "app" {
  name         = "my-app-project"
  display_name = "My Application"
}

# --- Step 2: Grant yourself owner access on the new project ---
# Without this, all resource operations on the new project will return 403.

resource "evroc_role_binding" "admin" {
  project   = evroc_project.app.name
  principal = "/iam/users/${var.admin_user_id}"

  roles {
    role = "/iam/roles/projectOwner"
  }
}

# --- Step 3: Create resources in the new project ---
# These depend on the role binding so Terraform waits for access to be granted.

data "evroc_disk_images" "available" {
  depends_on = [evroc_role_binding.admin]
}

data "evroc_compute_profiles" "available" {
  depends_on = [evroc_role_binding.admin]
}

resource "evroc_disk" "boot" {
  project = evroc_project.app.name
  name    = "app-boot-disk"
  size    = 20
  image   = data.evroc_disk_images.available.ubuntu_minimal_24_04_1
  zone    = "a"

  depends_on = [evroc_role_binding.admin]
}

resource "evroc_public_ip" "app" {
  project = evroc_project.app.name
  name    = "app-public-ip"

  depends_on = [evroc_role_binding.admin]
}

resource "evroc_security_group" "app" {
  project = evroc_project.app.name
  name    = "app-sg"

  rule {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-egress"
    direction = "Egress"
    protocol  = "All"
    port      = 0
    remote_ip = "0.0.0.0/0"
  }

  depends_on = [evroc_role_binding.admin]
}

resource "evroc_virtual_machine" "app" {
  project   = evroc_project.app.name
  name      = "app-server"
  flavor    = data.evroc_compute_profiles.available.a1a_s
  boot_disk = evroc_disk.boot.fqid
  zone      = "a"

  public_ip       = evroc_public_ip.app.fqid
  security_groups = [evroc_security_group.app.fqid]

  ssh_keys = [
    "ssh-ed25519 AAAAC3... user@example.com"
  ]
}

# --- Outputs ---

output "project_name" {
  value = evroc_project.app.name
}

output "vm_ip" {
  value = evroc_virtual_machine.app.public_ipv4_address
}

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

# Optional: Query available disk images and compute profiles
# Uncomment to use dynamic image/profile references:
#
# data "evroc_disk_images" "available" {}
# data "evroc_compute_profiles" "available" {}
#
# Then reference them like:
#   image  = data.evroc_disk_images.available.ubuntu_minimal_24_04_1
#   flavor = data.evroc_compute_profiles.available.a1a_s

# Create a public IP for the VM
resource "evroc_public_ip" "vm" {
  name = "test-vm-ip"
}

# Create a security group allowing SSH
resource "evroc_security_group" "allow_ssh" {
  name = "test-vm-allow-ssh"

  rule {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-all-tcp-egress"
    direction = "Egress"
    protocol  = "TCP"
    port      = 0 # 0 means all ports
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-all-udp-egress"
    direction = "Egress"
    protocol  = "UDP"
    port      = 0 # 0 means all ports
    remote_ip = "0.0.0.0/0"
  }
}

# Create a boot disk
resource "evroc_disk" "boot" {
  name  = "test-vm-boot-disk"
  size  = 20
  image = "ubuntu-minimal.24-04.1" # or use: data.evroc_disk_images.available.images[0]
  zone  = "a"
}

# Create a VM with public IP and security group
resource "evroc_virtual_machine" "test" {
  name      = "test-vm"
  flavor    = "a1a.s" # Or use: data.evroc_compute_profiles.available.profiles[X]
  boot_disk = evroc_disk.boot.fqid
  zone      = "a"

  # Attach public IP
  public_ip = evroc_public_ip.vm.fqid

  # Attach security group
  security_groups = [
    evroc_security_group.allow_ssh.fqid
  ]

  ssh_keys = [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFeENOwB0QwUEicJGrFxt44yiShgBWzANhpE/5gNw041"
  ]
}

# Outputs
output "vm_id" {
  description = "Virtual machine UUID"
  value       = evroc_virtual_machine.test.vm_id
}

output "vm_status" {
  description = "Virtual machine status"
  value       = evroc_virtual_machine.test.status
}

output "public_ip" {
  description = "Public IPv4 address"
  value       = evroc_virtual_machine.test.public_ipv4_address
}

output "private_ip" {
  description = "Private IPv4 address"
  value       = evroc_virtual_machine.test.private_ipv4_address
}

output "disk_id" {
  description = "Boot disk UUID"
  value       = evroc_disk.boot.disk_id
}

output "ssh_command" {
  description = "SSH command to connect to the VM"
  value       = evroc_virtual_machine.test.public_ipv4_address != null ? "ssh evroc-user@${evroc_virtual_machine.test.public_ipv4_address}" : "IP not yet assigned - run 'terraform refresh'"
}

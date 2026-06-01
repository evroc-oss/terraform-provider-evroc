terraform {
  required_providers {
    evroc = {
      source  = "github.com/evroc-oss/evroc"
    }
  }
}

provider "evroc" {
  # Credentials and context read from ~/.evroc/config.yaml by default.
  # See: https://github.com/evroc-oss/terraform-provider-evroc#1-authenticate-with-evroc
}

# Create a boot disk
resource "evroc_disk" "boot" {
  name  = "vm-boot-disk"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

# Create a data disk (no image, just empty storage)
resource "evroc_disk" "data" {
  name = "vm-data-disk"
  size = 100
  zone = "a"
}

# Create a virtual machine
resource "evroc_virtual_machine" "app" {
  name      = "app-server"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.boot.fqid
  zone      = "a"

  ssh_keys = [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... user@example.com"
  ]

  cloud_config_user_data = <<-EOF
    ## template: jinja
    #cloud-config
    ssh_pwauth: false
    users:
      - name: ubuntu
        lock_passwd: true
        sudo: ALL=(ALL) NOPASSWD:ALL
        shell: /bin/bash
    {% if public_ssh_keys %}
        ssh_authorized_keys:
    {% for pubkey in public_ssh_keys %}
          - {{ pubkey }}
    {% endfor %}
    {% endif %}

    runcmd:
      - apt-get update
      - apt-get install -y lvm2
  EOF
}

# Hot-attach the data disk to the VM
resource "evroc_hotswap_disk_attachment" "data_attachment" {
  name            = "app-data-attachment"
  virtual_machine = evroc_virtual_machine.app.fqid
  disk            = evroc_disk.data.fqid
}

# Query the disk attachment
data "evroc_hotswap_disk_attachment" "data_attachment" {
  name = evroc_hotswap_disk_attachment.data_attachment.name
}

# Output attachment details
output "attachment_id" {
  value       = evroc_hotswap_disk_attachment.data_attachment.attachment_id
  description = "ID of the disk attachment"
}

output "disk_serial" {
  value       = evroc_hotswap_disk_attachment.data_attachment.serial
  description = "Serial identifier of the attached disk"
}

output "attached_to_vm" {
  value       = data.evroc_hotswap_disk_attachment.data_attachment.virtual_machine
  description = "Name of the VM the disk is attached to"
}

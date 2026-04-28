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

# Create a placement group with spread strategy
resource "evroc_placement_group" "app_spread" {
  name     = "app-spread-pg"
  strategy = "spread"
  zone     = "a"
}

# Create disks for VMs
resource "evroc_disk" "app1" {
  name  = "app1-disk"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

resource "evroc_disk" "app2" {
  name  = "app2-disk"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

# Create VMs in the placement group
# Note: VM placement group attachment would be done via VM resource
resource "evroc_virtual_machine" "app1" {
  name      = "app-vm-1"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.app1.fqid
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
      - echo "VM 1 in placement group"
  EOF
}

resource "evroc_virtual_machine" "app2" {
  name      = "app-vm-2"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.app2.fqid
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
      - echo "VM 2 in placement group"
  EOF
}

# Query placement group information
data "evroc_placement_group" "app_spread" {
  name = evroc_placement_group.app_spread.name
}

# Output placement group details
output "placement_group_id" {
  value       = evroc_placement_group.app_spread.pg_id
  description = "ID of the placement group"
}

output "placement_group_strategy" {
  value       = data.evroc_placement_group.app_spread.strategy
  description = "Placement strategy of the group"
}

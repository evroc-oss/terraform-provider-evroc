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

# Create a boot disk
resource "evroc_disk" "boot" {
  name  = "vm-boot-disk"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

# Create a virtual machine
resource "evroc_virtual_machine" "example" {
  name      = "example-vm"
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
      - echo "Hello from evroc VM!" > /tmp/hello.txt
      - apt-get update
      - apt-get install -y nginx
  EOF
}

# Output VM details
output "vm_id" {
  value       = evroc_virtual_machine.example.vm_id
  description = "The unique ID of the virtual machine"
}

output "vm_status" {
  value       = evroc_virtual_machine.example.status
  description = "The current status of the VM"
}

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

# Create a security group with rules
resource "evroc_security_group" "web" {
  name = "web-sg"

  # Allow SSH
  rule {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }

  # Allow HTTP
  rule {
    name      = "allow-http"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 80
    remote_ip = "0.0.0.0/0"
  }

  # Allow HTTPS
  rule {
    name      = "allow-https"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 443
    remote_ip = "0.0.0.0/0"
  }

  # Allow all outbound
  rule {
    name      = "allow-all-egress"
    direction = "Egress"
    protocol  = "TCP"
    port      = 0
    remote_ip = "0.0.0.0/0"
  }
}

# Create a boot disk
resource "evroc_disk" "web_boot" {
  name  = "web-boot-disk"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

# Allocate a public IP
resource "evroc_public_ip" "web" {
  name = "web-public-ip"
}

# Create a VM with networking
resource "evroc_virtual_machine" "web" {
  name      = "web-server"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.web_boot.fqid
  zone      = "a"

  # Attach public IP
  public_ip = evroc_public_ip.web.fqid

  # Attach security group
  security_groups = [evroc_security_group.web.fqid]

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
      - apt-get install -y nginx
      - systemctl enable nginx
      - systemctl start nginx
      - echo "Hello from evroc!" > /var/www/html/index.html
  EOF
}

# Output the public IP address
output "web_public_ip" {
  value       = evroc_public_ip.web.ip_address
  description = "Public IP address of the web server"
}

output "security_group_id" {
  value       = evroc_security_group.web.sg_id
  description = "ID of the security group"
}

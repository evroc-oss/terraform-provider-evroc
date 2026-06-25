terraform {
  required_providers {
    evroc = {
      source = "registry.terraform.io/evroc/evroc"
    }
  }
}

variable "run_id" {
  type        = string
  description = "Unique suffix for resource names."
}

variable "ssh_public_key" {
  type        = string
  description = "SSH public key for e2e test VMs."
}

locals {
  ts = var.run_id
  common_labels = {
    "managed-by" = "terraform"
    "test-run"   = var.run_id
  }
}

# Uses project/org/region from ~/.evroc/config.yaml
provider "evroc" {}

data "evroc_disk_images" "images" {}
data "evroc_compute_profiles" "profiles" {}

# ── VPC / Subnet ────────────────────────────

resource "evroc_vpc" "main" {
  name             = "e2e-vpc-${local.ts}"
  ipv4_cidr_blocks = ["10.100.0.0/16"]
  user_labels      = merge(local.common_labels, { "component" = "networking" })
}

resource "evroc_subnet" "app" {
  name            = "e2e-subnet-${local.ts}"
  vpc_ref         = evroc_vpc.main.fqid
  zone            = "a"
  ipv4_cidr_block = "10.100.1.0/24"
  user_labels     = merge(local.common_labels, { "component" = "networking" })
}

# ── Networking ───────────────────────────────

resource "evroc_placement_group" "spread" {
  name        = "e2e-pg-${local.ts}"
  strategy    = "spread"
  zone        = "a"
  user_labels = merge(local.common_labels, { "component" = "compute" })
}

resource "evroc_security_group" "web" {
  name    = "e2e-sg-${local.ts}"
  vpc_ref = evroc_vpc.main.fqid

  rule {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-http"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 80
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-egress-tcp"
    direction = "Egress"
    protocol  = "TCP"
    port      = 0
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-egress-udp"
    direction = "Egress"
    protocol  = "UDP"
    port      = 0
    remote_ip = "0.0.0.0/0"
  }

  user_labels = merge(local.common_labels, { "component" = "networking" })
}

resource "evroc_public_ip" "vm" {
  name        = "e2e-pip-${local.ts}"
  user_labels = merge(local.common_labels, { "component" = "networking" })
}

resource "evroc_public_ip" "lb" {
  name        = "e2e-lb-pip-${local.ts}"
  user_labels = merge(local.common_labels, { "component" = "networking" })
}

# ── Compute ──────────────────────────────────

resource "evroc_disk" "boot" {
  name        = "e2e-boot-${local.ts}"
  size        = 20
  image       = data.evroc_disk_images.images.ubuntu_minimal_24_04_1
  zone        = "a"
  user_labels = merge(local.common_labels, { "component" = "compute" })
}

resource "evroc_disk" "data" {
  name        = "e2e-data-${local.ts}"
  size        = 10
  zone        = "a"
  user_labels = merge(local.common_labels, { "component" = "storage" })
}

resource "evroc_virtual_machine" "web" {
  name   = "e2e-vm-${local.ts}"
  flavor = data.evroc_compute_profiles.profiles.a1a_xs
  zone   = "a"

  boot_disk       = evroc_disk.boot.fqid
  public_ip       = evroc_public_ip.vm.fqid
  security_groups = [evroc_security_group.web.fqid]
  placement_group = evroc_placement_group.spread.fqid
  subnet_ref      = evroc_subnet.app.fqid

  ssh_keys = [
    var.ssh_public_key
  ]

  cloud_config_user_data = <<-EOF
    #cloud-config
    package_update: true
    packages:
      - nginx
  EOF

  user_labels = merge(local.common_labels, { "component" = "compute" })
}

resource "evroc_hotswap_disk_attachment" "data" {
  name            = "e2e-attach-${local.ts}"
  virtual_machine = evroc_virtual_machine.web.name
  disk            = evroc_disk.data.name
  user_labels     = merge(local.common_labels, { "component" = "storage" })
}

# ── Load Balancer ────────────────────────────

resource "evroc_lb_backend_pool" "app" {
  name         = "e2e-pool-${local.ts}"
  backend_refs = [evroc_virtual_machine.web.fqid]
  user_labels  = merge(local.common_labels, { "component" = "networking" })
}

resource "evroc_lb_backend_service" "http" {
  name             = "e2e-bsvc-${local.ts}"
  port             = 80
  backend_pool_ref = evroc_lb_backend_pool.app.fqid
  user_labels      = merge(local.common_labels, { "component" = "networking" })
}

resource "evroc_lb_l4_route" "http" {
  name                        = "e2e-route-${local.ts}"
  default_backend_service_ref = evroc_lb_backend_service.http.fqid
  user_labels                 = merge(local.common_labels, { "component" = "networking" })
}

resource "evroc_loadbalancer" "app" {
  name          = "e2e-lb-${local.ts}"
  public_ip_ref = evroc_public_ip.lb.fqid

  listener {
    name       = "http"
    protocol   = "TCP"
    port       = 80
    route_refs = [evroc_lb_l4_route.http.fqid]
  }

  user_labels = merge(local.common_labels, { "component" = "networking" })
}

# ── Data sources for new resources ───────────

data "evroc_vpc" "main" {
  name = evroc_vpc.main.name
}

data "evroc_subnet" "app" {
  name = evroc_subnet.app.name
}

data "evroc_loadbalancer" "app" {
  name = evroc_loadbalancer.app.name
}

data "evroc_lb_backend_pool" "app" {
  name = evroc_lb_backend_pool.app.name
}

data "evroc_lb_backend_service" "http" {
  name = evroc_lb_backend_service.http.name
}

data "evroc_lb_l4_route" "http" {
  name = evroc_lb_l4_route.http.name
}

# ── Storage ──────────────────────────────────

resource "evroc_bucket" "assets" {
  name                  = "e2e-bucket-${local.ts}"
  object_retention_mode = "Disabled"
  user_labels           = merge(local.common_labels, { "component" = "storage" })
}

resource "evroc_bucket" "logs" {
  name                  = "e2e-logs-${local.ts}"
  object_retention_mode = "Versioned"
  user_labels           = merge(local.common_labels, { "component" = "storage" })
}

resource "evroc_bucket_service_account" "sa" {
  name        = "e2e-sa-${local.ts}"
  buckets     = [evroc_bucket.assets.name, evroc_bucket.logs.name]
  user_labels = merge(local.common_labels, { "component" = "storage" })
}

# ── Outputs ──────────────────────────────────

output "vm_ip" {
  value = evroc_virtual_machine.web.public_ipv4_address
}

output "vm_private_ip" {
  value = evroc_virtual_machine.web.private_ipv4_address
}

output "vm_status" {
  value = evroc_virtual_machine.web.status
}

output "bucket" {
  value = evroc_bucket.assets.name
}

output "bucket_logs" {
  value = evroc_bucket.logs.name
}

output "lb_ip" {
  value = evroc_loadbalancer.app.public_ipv4_address
}

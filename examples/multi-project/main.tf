terraform {
  required_providers {
    evroc = {
      source = "github.com/evroc-oss/evroc"
    }
  }
}

# -------------------------------------------------------------------------
# Provider aliases — one per project.
#
# Each alias targets a different project so resources declared with
# `provider = evroc.<alias>` are created in the correct project scope.
# Credentials are shared (same token); only the project ID changes.
# -------------------------------------------------------------------------

provider "evroc" {
  alias        = "shared"
  project      = var.shared_project_id
  organization = var.organization_id
  region       = var.region
  # Auth credentials read from ~/.evroc/config.yaml
}

provider "evroc" {
  alias        = "dev"
  project      = var.dev_project_id
  organization = var.organization_id
  region       = var.region
}

provider "evroc" {
  alias        = "prod"
  project      = var.prod_project_id
  organization = var.organization_id
  region       = var.region
}

# -------------------------------------------------------------------------
# Shared security group rule definitions.
#
# These locals are the single source of truth for firewall policy. The same
# rules are stamped into each project below, so every environment enforces
# the exact same policy without copy-pasting rule blocks.
# -------------------------------------------------------------------------

locals {
  rules_ssh_ingress = {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }

  rules_http_ingress = {
    name      = "allow-http"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 80
    remote_ip = "0.0.0.0/0"
  }

  rules_https_ingress = {
    name      = "allow-https"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 443
    remote_ip = "0.0.0.0/0"
  }

  rules_tcp_egress = {
    name      = "allow-all-tcp-egress"
    direction = "Egress"
    protocol  = "TCP"
    port      = 0
    remote_ip = "0.0.0.0/0"
  }

  rules_udp_egress = {
    name      = "allow-all-udp-egress"
    direction = "Egress"
    protocol  = "UDP"
    port      = 0
    remote_ip = "0.0.0.0/0"
  }
}

# -------------------------------------------------------------------------
# Shared project — canonical security group definitions.
#
# The "shared" project acts as the source of record for security policy.
# No workload VMs run here; it only holds the authoritative group objects
# that other projects mirror.
# -------------------------------------------------------------------------

resource "evroc_security_group" "shared_web" {
  provider = evroc.shared
  name     = "standard-web"

  rule {
    name      = local.rules_ssh_ingress.name
    direction = local.rules_ssh_ingress.direction
    protocol  = local.rules_ssh_ingress.protocol
    port      = local.rules_ssh_ingress.port
    remote_ip = local.rules_ssh_ingress.remote_ip
  }

  rule {
    name      = local.rules_http_ingress.name
    direction = local.rules_http_ingress.direction
    protocol  = local.rules_http_ingress.protocol
    port      = local.rules_http_ingress.port
    remote_ip = local.rules_http_ingress.remote_ip
  }

  rule {
    name      = local.rules_https_ingress.name
    direction = local.rules_https_ingress.direction
    protocol  = local.rules_https_ingress.protocol
    port      = local.rules_https_ingress.port
    remote_ip = local.rules_https_ingress.remote_ip
  }

  rule {
    name      = local.rules_tcp_egress.name
    direction = local.rules_tcp_egress.direction
    protocol  = local.rules_tcp_egress.protocol
    port      = local.rules_tcp_egress.port
    remote_ip = local.rules_tcp_egress.remote_ip
  }

  rule {
    name      = local.rules_udp_egress.name
    direction = local.rules_udp_egress.direction
    protocol  = local.rules_udp_egress.protocol
    port      = local.rules_udp_egress.port
    remote_ip = local.rules_udp_egress.remote_ip
  }
}

# -------------------------------------------------------------------------
# Dev project — mirrors the shared security group, then deploys a VM.
#
# Security groups are project-scoped, so each project needs its own copy.
# The copy is created from the same `locals` block, not duplicated by hand,
# which guarantees the dev and prod policies stay in sync with `shared`.
# -------------------------------------------------------------------------

resource "evroc_security_group" "dev_web" {
  provider = evroc.dev

  # Name matches the shared project's group — easy to cross-reference in
  # dashboards and audit logs.
  name = evroc_security_group.shared_web.name

  rule {
    name      = local.rules_ssh_ingress.name
    direction = local.rules_ssh_ingress.direction
    protocol  = local.rules_ssh_ingress.protocol
    port      = local.rules_ssh_ingress.port
    remote_ip = local.rules_ssh_ingress.remote_ip
  }

  rule {
    name      = local.rules_http_ingress.name
    direction = local.rules_http_ingress.direction
    protocol  = local.rules_http_ingress.protocol
    port      = local.rules_http_ingress.port
    remote_ip = local.rules_http_ingress.remote_ip
  }

  rule {
    name      = local.rules_https_ingress.name
    direction = local.rules_https_ingress.direction
    protocol  = local.rules_https_ingress.protocol
    port      = local.rules_https_ingress.port
    remote_ip = local.rules_https_ingress.remote_ip
  }

  rule {
    name      = local.rules_tcp_egress.name
    direction = local.rules_tcp_egress.direction
    protocol  = local.rules_tcp_egress.protocol
    port      = local.rules_tcp_egress.port
    remote_ip = local.rules_tcp_egress.remote_ip
  }

  rule {
    name      = local.rules_udp_egress.name
    direction = local.rules_udp_egress.direction
    protocol  = local.rules_udp_egress.protocol
    port      = local.rules_udp_egress.port
    remote_ip = local.rules_udp_egress.remote_ip
  }
}

resource "evroc_public_ip" "dev_vm" {
  provider = evroc.dev
  name     = "dev-web-ip"
}

resource "evroc_disk" "dev_boot" {
  provider = evroc.dev
  name     = "dev-web-boot"
  size     = 20
  image    = "ubuntu-minimal.24-04.1"
  zone     = "a"
}

resource "evroc_virtual_machine" "dev_web" {
  provider  = evroc.dev
  name      = "dev-web"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.dev_boot.fqid
  zone      = "a"

  public_ip = evroc_public_ip.dev_vm.fqid

  # Reference the dev project's security group by FQID.
  # The name was taken from the shared group, so it is consistent across projects.
  security_groups = [evroc_security_group.dev_web.fqid]

  ssh_keys = [var.ssh_public_key]
}

# -------------------------------------------------------------------------
# Prod project — same pattern: mirror the security group, deploy a VM.
#
# Because both dev and prod are derived from the same `locals`, a future
# policy change (e.g. locking down SSH to a bastion CIDR) only requires
# editing the local value once — both projects update on the next apply.
# -------------------------------------------------------------------------

resource "evroc_security_group" "prod_web" {
  provider = evroc.prod
  name     = evroc_security_group.shared_web.name

  rule {
    name      = local.rules_ssh_ingress.name
    direction = local.rules_ssh_ingress.direction
    protocol  = local.rules_ssh_ingress.protocol
    port      = local.rules_ssh_ingress.port
    remote_ip = local.rules_ssh_ingress.remote_ip
  }

  rule {
    name      = local.rules_http_ingress.name
    direction = local.rules_http_ingress.direction
    protocol  = local.rules_http_ingress.protocol
    port      = local.rules_http_ingress.port
    remote_ip = local.rules_http_ingress.remote_ip
  }

  rule {
    name      = local.rules_https_ingress.name
    direction = local.rules_https_ingress.direction
    protocol  = local.rules_https_ingress.protocol
    port      = local.rules_https_ingress.port
    remote_ip = local.rules_https_ingress.remote_ip
  }

  rule {
    name      = local.rules_tcp_egress.name
    direction = local.rules_tcp_egress.direction
    protocol  = local.rules_tcp_egress.protocol
    port      = local.rules_tcp_egress.port
    remote_ip = local.rules_tcp_egress.remote_ip
  }

  rule {
    name      = local.rules_udp_egress.name
    direction = local.rules_udp_egress.direction
    protocol  = local.rules_udp_egress.protocol
    port      = local.rules_udp_egress.port
    remote_ip = local.rules_udp_egress.remote_ip
  }
}

resource "evroc_public_ip" "prod_vm" {
  provider = evroc.prod
  name     = "prod-web-ip"
}

resource "evroc_disk" "prod_boot" {
  provider = evroc.prod
  name     = "prod-web-boot"
  size     = 50
  image    = "ubuntu-minimal.24-04.1"
  zone     = "a"
}

resource "evroc_virtual_machine" "prod_web" {
  provider  = evroc.prod
  name      = "prod-web"
  flavor    = "a1a.m"
  boot_disk = evroc_disk.prod_boot.fqid
  zone      = "a"

  public_ip = evroc_public_ip.prod_vm.fqid

  security_groups = [evroc_security_group.prod_web.fqid]

  ssh_keys = [var.ssh_public_key]
}

# -------------------------------------------------------------------------
# Lookup example — read an existing security group back from any project.
#
# If a security group already existed before this Terraform config (e.g.
# created manually or by another config), use a data source with the correct
# provider alias to bring it into scope without managing its lifecycle.
# -------------------------------------------------------------------------

data "evroc_security_group" "shared_web_lookup" {
  provider = evroc.shared
  name     = evroc_security_group.shared_web.name

  depends_on = [evroc_security_group.shared_web]
}

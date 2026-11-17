terraform {
  required_providers {
    evroc = {
      source = "evroc/evroc"
    }
  }
}

provider "evroc" {
  project = var.project
  region  = var.region
}

variable "project" {
  type        = string
  description = "Project containing the backend VMs and load balancer."
}

variable "region" {
  type        = string
  description = "Region containing the backend VMs and load balancer."
}

variable "backend_refs" {
  type        = set(string)
  description = "Fully qualified references to VMs serving HTTP on port 8080 in the default VPC."
}

variable "hostname" {
  type        = string
  description = "Application hostname covered by the certificate."
}

variable "certificate_file" {
  type        = string
  description = "Path to the PEM certificate chain, leaf first, followed by intermediates without the root CA."
}

variable "private_key_file" {
  type        = string
  description = "Path to the matching PEM private key."
}

resource "evroc_public_ip" "web" {
  name = "web-ip"
}

resource "evroc_lb_backend_pool" "web" {
  name         = "web-pool"
  backend_refs = var.backend_refs
}

resource "evroc_lb_backend_service" "web" {
  name                  = "web-backend"
  port                  = 8080
  backend_pool_ref      = evroc_lb_backend_pool.web.fqid
  ip_protocol_selection = "IPv4"

  health_check {
    http {
      path = "/healthz"
    }
  }
}

resource "evroc_lb_http_route" "web" {
  name      = "web-route"
  hostnames = [var.hostname]

  rule {
    name                = "web"
    backend_service_ref = evroc_lb_backend_service.web.fqid
  }
}

resource "evroc_lb_certificate" "web" {
  name            = "web-cert"
  certificate_pem = file(var.certificate_file)
  private_key_pem = file(var.private_key_file)
}

resource "evroc_loadbalancer" "web" {
  name          = "web-lb"
  public_ip_ref = evroc_public_ip.web.fqid

  listener {
    name       = "https"
    protocol   = "HTTPS"
    port       = 443
    route_refs = [evroc_lb_http_route.web.fqid]
    tls {
      certificate_ref = evroc_lb_certificate.web.fqid
    }
  }
}

output "public_ipv4_address" {
  value = evroc_loadbalancer.web.public_ipv4_address
}

output "certificate_expiry" {
  value = evroc_lb_certificate.web.not_after
}

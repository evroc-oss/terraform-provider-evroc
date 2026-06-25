terraform {
  required_providers {
    evroc = {
      source = "registry.terraform.io/evroc/evroc"
    }
  }
}

provider "evroc" {}

# --- Security Group ---
resource "evroc_security_group" "k3s" {
  name    = "k3s-ha-sg"
  vpc_ref = "default-se-sto"

  rule {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-k8s-api"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 6443
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-k3s-supervisor"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 9345
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-etcd-client"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 2379
    remote_ip = var.cluster_cidr
  }

  rule {
    name      = "allow-etcd-peer"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 2380
    remote_ip = var.cluster_cidr
  }

  rule {
    name      = "allow-kubelet"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 10250
    remote_ip = var.cluster_cidr
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
}

# --- Load Balancer: the public Kubernetes API endpoint ---
resource "evroc_public_ip" "k8s_api" {
  name = "k3s-api-lb-ip"
}

resource "evroc_lb_backend_pool" "k8s_api" {
  name         = "k3s-api-pool"
  backend_refs = concat([evroc_virtual_machine.control_plane_first.fqid], [for vm in evroc_virtual_machine.control_plane_rest : vm.fqid])

  user_labels = {
    cluster = "k3s-ha"
    role    = "control-plane"
  }
}

resource "evroc_lb_backend_service" "k8s_api" {
  name             = "k3s-api-svc"
  port             = 6443
  backend_pool_ref = evroc_lb_backend_pool.k8s_api.fqid

  health_check {
    interval            = "10s"
    timeout             = "5s"
    healthy_threshold   = 2
    unhealthy_threshold = 3

    tcp {}
  }

  user_labels = {
    cluster = "k3s-ha"
    role    = "control-plane"
  }
}

resource "evroc_lb_l4_route" "k8s_api" {
  name                        = "k3s-api-route"
  default_backend_service_ref = evroc_lb_backend_service.k8s_api.fqid

  user_labels = {
    cluster = "k3s-ha"
    role    = "control-plane"
  }
}

resource "evroc_loadbalancer" "k8s_api" {
  name          = "k3s-api-lb"
  public_ip_ref = evroc_public_ip.k8s_api.fqid

  listener {
    name       = "k8s-api"
    protocol   = "TCP"
    port       = 6443
    route_refs = [evroc_lb_l4_route.k8s_api.fqid]
  }

  user_labels = {
    cluster = "k3s-ha"
    role    = "control-plane"
  }
}

# --- First control plane ---
resource "evroc_disk" "control_plane_first" {
  name  = "k3s-cp-1-boot"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = var.zones[0]
}

resource "evroc_virtual_machine" "control_plane_first" {
  name      = "k3s-cp-1"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.control_plane_first.fqid
  zone      = var.zones[0]

  subnet_ref      = "default-se-sto-${var.zones[0]}"
  security_groups = [evroc_security_group.k3s.fqid]
  ssh_keys        = [var.ssh_public_key]

  cloud_config_user_data = templatefile("${path.module}/cloud-init-server.yaml", {
    hostname   = "k3s-cp-1"
    k3s_token  = var.k3s_token
    lb_host    = evroc_public_ip.k8s_api.ip_address
    server_url = ""
    is_first   = true
  })
}

# --- Remaining control planes ---
resource "evroc_disk" "control_plane_rest" {
  count = var.control_plane_count - 1

  name  = "k3s-cp-${count.index + 2}-boot"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = var.zones[(count.index + 1) % length(var.zones)]
}

resource "evroc_virtual_machine" "control_plane_rest" {
  count = var.control_plane_count - 1

  name      = "k3s-cp-${count.index + 2}"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.control_plane_rest[count.index].fqid
  zone      = var.zones[(count.index + 1) % length(var.zones)]

  subnet_ref      = "default-se-sto-${var.zones[(count.index + 1) % length(var.zones)]}"
  security_groups = [evroc_security_group.k3s.fqid]
  ssh_keys        = [var.ssh_public_key]

  cloud_config_user_data = templatefile("${path.module}/cloud-init-server.yaml", {
    hostname   = "k3s-cp-${count.index + 2}"
    k3s_token  = var.k3s_token
    lb_host    = evroc_public_ip.k8s_api.ip_address
    server_url = "https://${evroc_virtual_machine.control_plane_first.private_ipv4_address}:6443"
    is_first   = false
  })

  depends_on = [evroc_virtual_machine.control_plane_first]
}

# --- Workers ---
resource "evroc_disk" "worker" {
  count = var.worker_count

  name  = "k3s-worker-${count.index + 1}-boot"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = var.zones[count.index % length(var.zones)]
}

resource "evroc_virtual_machine" "worker" {
  count = var.worker_count

  name      = "k3s-worker-${count.index + 1}"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.worker[count.index].fqid
  zone      = var.zones[count.index % length(var.zones)]

  subnet_ref      = "default-se-sto-${var.zones[count.index % length(var.zones)]}"
  security_groups = [evroc_security_group.k3s.fqid]
  ssh_keys        = [var.ssh_public_key]

  cloud_config_user_data = templatefile("${path.module}/cloud-init-agent.yaml", {
    hostname       = "k3s-worker-${count.index + 1}"
    k3s_token      = var.k3s_token
    k3s_server_url = "https://${evroc_virtual_machine.control_plane_first.private_ipv4_address}:6443"
  })

  depends_on = [evroc_virtual_machine.control_plane_first]
}

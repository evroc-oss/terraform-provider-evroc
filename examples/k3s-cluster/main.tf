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

# Security group for k3s cluster
resource "evroc_security_group" "k3s" {
  name = "k3s-cluster-sg"

  # SSH access
  rule {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }

  # Kubernetes API server
  rule {
    name      = "allow-k8s-api"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 6443
    remote_ip = "0.0.0.0/0"
  }

  # HTTP for ingress
  rule {
    name      = "allow-http"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 80
    remote_ip = "0.0.0.0/0"
  }

  # HTTPS for ingress
  rule {
    name      = "allow-https"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 443
    remote_ip = "0.0.0.0/0"
  }

  # k3s supervisor port (for worker nodes)
  rule {
    name      = "allow-k3s-supervisor"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 9345
    remote_ip = "0.0.0.0/0"
  }

  # Allow all outbound traffic
  rule {
    name      = "allow-all-tcp-egress"
    direction = "Egress"
    protocol  = "TCP"
    port      = 0
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-all-udp-egress"
    direction = "Egress"
    protocol  = "UDP"
    port      = 0
    remote_ip = "0.0.0.0/0"
  }
}

# ====================================
# Control Plane Node
# ====================================

resource "evroc_disk" "control_plane_boot" {
  name  = "k3s-control-plane-boot"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

resource "evroc_public_ip" "control_plane" {
  name = "k3s-control-plane-ip"
}

resource "evroc_virtual_machine" "control_plane" {
  name      = "k3s-control-plane"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.control_plane_boot.fqid
  zone      = "a"

  public_ip       = evroc_public_ip.control_plane.fqid
  security_groups = [evroc_security_group.k3s.fqid]

  ssh_keys = [
    var.ssh_public_key
  ]

  cloud_config_user_data = templatefile("${path.module}/cloud-init-server.yaml", {
    hostname  = "k3s-control-plane"
    k3s_token = var.k3s_token
    public_ip = evroc_public_ip.control_plane.ip_address
  })
}

# ====================================
# Worker Node 1
# ====================================

resource "evroc_disk" "worker1_boot" {
  name  = "k3s-worker1-boot"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

resource "evroc_public_ip" "worker1" {
  name = "k3s-worker1-ip"
}

resource "evroc_virtual_machine" "worker1" {
  name      = "k3s-worker1"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.worker1_boot.fqid
  zone      = "a"

  public_ip       = evroc_public_ip.worker1.fqid
  security_groups = [evroc_security_group.k3s.fqid]

  ssh_keys = [
    var.ssh_public_key
  ]

  cloud_config_user_data = templatefile("${path.module}/cloud-init-agent.yaml", {
    hostname       = "k3s-worker1"
    k3s_token      = var.k3s_token
    k3s_server_url = "https://${evroc_public_ip.control_plane.ip_address}:6443"
  })

  depends_on = [
    evroc_virtual_machine.control_plane
  ]
}

# ====================================
# Worker Node 2
# ====================================

resource "evroc_disk" "worker2_boot" {
  name  = "k3s-worker2-boot"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

resource "evroc_public_ip" "worker2" {
  name = "k3s-worker2-ip"
}

resource "evroc_virtual_machine" "worker2" {
  name      = "k3s-worker2"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.worker2_boot.fqid
  zone      = "a"

  public_ip       = evroc_public_ip.worker2.fqid
  security_groups = [evroc_security_group.k3s.fqid]

  ssh_keys = [
    var.ssh_public_key
  ]

  cloud_config_user_data = templatefile("${path.module}/cloud-init-agent.yaml", {
    hostname       = "k3s-worker2"
    k3s_token      = var.k3s_token
    k3s_server_url = "https://${evroc_public_ip.control_plane.ip_address}:6443"
  })

  depends_on = [
    evroc_virtual_machine.control_plane
  ]
}

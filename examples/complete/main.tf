terraform {
  required_providers {
    evroc = {
      source  = "github.com/evroc-oss/evroc"
      version = "~> 0.1"
    }
  }
}

provider "evroc" {
  project = var.project
  region  = var.region
}

# Create a boot disk for the web server
resource "evroc_disk" "web_boot" {
  name  = "${var.prefix}-web-boot"
  size  = 50
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

# Create a data disk for the web server
resource "evroc_disk" "web_data" {
  name = "${var.prefix}-web-data"
  size = 100
  zone = "a"
}

# Allocate a public IP for the web server
resource "evroc_public_ip" "web" {
  name = "${var.prefix}-web-ip"
}

# Create the web server VM
resource "evroc_virtual_machine" "web" {
  name      = "${var.prefix}-web-server"
  flavor    = var.vm_flavor
  boot_disk = evroc_disk.web_boot.fqid
  zone      = "a"

  public_ip = evroc_public_ip.web.fqid
  ssh_keys  = var.ssh_keys

  cloud_config_user_data = templatefile("${path.module}/cloud-init.yaml", {
    hostname = "${var.prefix}-web-server"
  })
}

# Data source example - look up an existing disk
data "evroc_disk" "existing" {
  count  = var.lookup_existing_disk ? 1 : 0
  name   = var.existing_disk_name
  region = var.region
}

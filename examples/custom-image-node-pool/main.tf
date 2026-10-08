terraform {
  required_providers {
    evroc = {
      source = "evroc-oss/evroc"
    }
  }
}

provider "evroc" {}

# The platform team bakes the node image out of band (qemu-img/virt-customize
# or Packer), names it after the Kubernetes version it carries, e.g.
# k8s-node-ubuntu2404-v1.35.0, and uploads it to the versioned bucket below.
# Everything after the upload is owned by Terraform: the bucket, the service
# account that reads it, the registration pinned to one object version, and
# the disks booted from it.
resource "evroc_bucket" "images" {
  name                  = "${var.pool_name}-images"
  object_retention_mode = "Versioned"
}

resource "evroc_bucket_service_account" "image_reader" {
  name    = "${var.pool_name}-image-reader"
  buckets = [evroc_bucket.images.name]
}

resource "evroc_custom_disk_image" "node" {
  name                   = var.node_image_name
  bucket                 = evroc_bucket.images.fqid
  bucket_service_account = evroc_bucket_service_account.image_reader.fqid
  object_path            = var.node_image_object
  object_version         = var.node_image_object_version
  default_disk_size      = var.node_disk_size
  os_name                = "ubuntu"
  os_version             = "24.04"
  image_version          = var.node_image_name
}

resource "evroc_disk" "node_boot" {
  count = var.node_count

  name  = "${var.pool_name}-${count.index}-boot"
  zone  = var.zone
  size  = evroc_custom_disk_image.node.default_disk_size
  image = evroc_custom_disk_image.node.fqid
}

resource "evroc_virtual_machine" "node" {
  count = var.node_count

  name      = "${var.pool_name}-${count.index}"
  flavor    = var.node_flavor
  zone      = var.zone
  boot_disk = evroc_disk.node_boot[count.index].fqid
  ssh_keys  = [var.ssh_public_key]

  security_groups = var.security_group_fqids

  # kubeadm, kubelet and containerd are already in the image, so bootstrap is
  # reduced to joining the cluster.
  cloud_config_user_data = templatefile("${path.module}/cloud-init.yaml", {
    hostname     = "${var.pool_name}-${count.index}"
    join_command = var.kubeadm_join_command
  })

  user_labels = {
    "k8s-pool"  = var.pool_name
    "k8s-image" = var.node_image_name
  }
}

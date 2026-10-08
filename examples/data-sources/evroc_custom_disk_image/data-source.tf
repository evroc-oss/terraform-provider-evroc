data "evroc_custom_disk_image" "node" {
  name = "ubuntu-k8s-v1"
}

resource "evroc_disk" "boot" {
  name  = "boot"
  zone  = "a"
  size  = 50 # At least the image's virtual disk size.
  image = data.evroc_custom_disk_image.node.fqid
}

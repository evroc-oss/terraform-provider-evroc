resource "evroc_bucket" "images" {
  name                  = "images"
  object_retention_mode = "Versioned"
}

resource "evroc_bucket_service_account" "image_reader" {
  name    = "image-reader"
  buckets = [evroc_bucket.images.name]
}

# Upload the image first, e.g.
#   aws s3 cp node-v1.qcow2 s3://images/images/node-v1.qcow2
#   aws s3api head-object --bucket images --key images/node-v1.qcow2 --query VersionId
resource "evroc_custom_disk_image" "node" {
  name                   = "node-v1"
  bucket                 = evroc_bucket.images.fqid
  bucket_service_account = evroc_bucket_service_account.image_reader.fqid
  object_path            = "images/node-v1.qcow2"
  object_version         = var.image_object_version
  default_disk_size      = 50 # At least the image's virtual disk size.
  os_name                = "ubuntu"
  os_version             = "24.04"
}

resource "evroc_disk" "boot" {
  name  = "boot"
  zone  = "a"
  size  = evroc_custom_disk_image.node.default_disk_size
  image = evroc_custom_disk_image.node.fqid
}

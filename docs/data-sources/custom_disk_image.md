---
page_title: "evroc_custom_disk_image Data Source - evroc"
subcategory: ""
description: |-
  Looks up a registered custom disk image by name in the provider region.
---

# evroc_custom_disk_image (Data Source)

Looks up an existing custom disk image by name. The image must already be registered.
A missing image fails the lookup. This data source does not build or publish images.

## Example Usage

```terraform
data "evroc_custom_disk_image" "node" {
  name = "ubuntu-k8s-v1"
}

resource "evroc_disk" "boot" {
  name  = "boot"
  zone  = "a"
  size  = 50 # At least the image's virtual disk size.
  image = data.evroc_custom_disk_image.node.fqid
}
```

The source attributes show what the registration points at. Add a `precondition` on
`object_version != ""` if your policy requires pinned images.

Project and region default to the provider configuration. Override `project` to
look up an image in another project; use a provider alias for another region.
The disk consuming the image must use the same project and region.
Choose a versioned image name explicitly; changing the disk's image replaces it.

## Schema

### Required

- `name` (String) Name of the registered custom disk image.

### Optional

- `project` (String) Project containing the image. Defaults to the provider project.

### Read-Only

- `architecture` (String) CPU architecture of the image.
- `bucket` (String) FQID of the bucket holding the image object.
- `bucket_service_account` (String) FQID of the bucket service account that reads the object.
- `default_disk_size` (Number) Default disk size in GB for disks created from this image.
- `description` (String) Description of the image.
- `fqid` (String) Fully qualified image reference. Pass this to `evroc_disk.image`.
- `id` (String) Fully qualified image reference.
- `image_version` (String) Descriptive version of the image build.
- `object_path` (String) Path of the image object within the bucket.
- `object_version` (String) Object version the registration is pinned to. Empty when unpinned, in which case new disks use whatever is at object_path.
- `os_name` (String) Operating system name.
- `os_version` (String) Operating system version.
- `ready` (Boolean) Whether the registration is ready for use.
- `region` (String) Region containing the image, selected by the provider configuration.

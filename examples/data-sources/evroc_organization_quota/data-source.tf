data "evroc_organization_quota" "this" {}
data "evroc_compute_profiles" "all" {}

variable "compute_profile" {
  type    = string
  default = "gn-l40s.s"
}

locals {
  profile = one([for p in data.evroc_compute_profiles.all.details : p if p.name == var.compute_profile])

  gpus_available = (
    lookup(data.evroc_organization_quota.this.compute_gpus, local.profile.gpu_model, 0) -
    lookup(data.evroc_organization_quota.this.usage_gpus, local.profile.gpu_model, 0)
  )
}

# Check GPU quota on the boot disk so a denied VM create doesn't leave the disk behind.
resource "evroc_disk" "boot" {
  name  = "gpu-vm-boot"
  size  = 100
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"

  lifecycle {
    precondition {
      condition     = local.gpus_available >= local.profile.gpu_quantity
      error_message = "Not enough ${local.profile.gpu_model} GPU quota for ${var.compute_profile}: ${local.gpus_available} available, ${local.profile.gpu_quantity} needed."
    }
  }
}

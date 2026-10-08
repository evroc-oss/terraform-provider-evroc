# Kubernetes node pool from a custom image

A platform team bakes a node image with containerd, kubelet and kubeadm
preinstalled, names it after the Kubernetes version it carries (for example
`k8s-node-ubuntu2404-v1.35.0`), and uploads it to a bucket. Terraform owns
everything from there: the bucket, the service account that reads it, the image
registration, and the node pool booted from it.

## What it shows

- The whole chain is Terraform references: `evroc_bucket` →
  `evroc_bucket_service_account` → `evroc_custom_disk_image` → `evroc_disk` →
  `evroc_virtual_machine`. Ordering, replacement and deletion follow the graph.
- The registration is pinned to one object version (`object_version`), so the
  image cannot change under a stable name.
- Disks take their size from the image's `default_disk_size`.
- `cloud-init` is reduced to the `kubeadm join`, because everything else is in
  the image.

Teams that only consume an image published by another workspace use the
`evroc_custom_disk_image` data source instead of the resource.

## Usage

1. Apply once to create the bucket and service account (the registration will
   fail until the object exists, so target them first):

   ```sh
   cp terraform.tfvars.example terraform.tfvars   # fill in your values
   terraform init
   terraform apply -target=evroc_bucket.images -target=evroc_bucket_service_account.image_reader
   ```

2. Build the image (see the SDK's custom disk image guide for the
   `virt-customize`/`virt-sysprep` steps) and upload it, capturing the version:

   ```sh
   aws s3 cp k8s-node-ubuntu2404-v1.35.0.qcow2 \
     s3://$(terraform output -raw image_bucket)/images/k8s-node-ubuntu2404-v1.35.0.qcow2
   aws s3api head-object --bucket $(terraform output -raw image_bucket) \
     --key images/k8s-node-ubuntu2404-v1.35.0.qcow2 --query VersionId --output text
   ```

3. Put the path and version ID in `terraform.tfvars` and apply the rest:

   ```sh
   terraform apply
   ```

## Rolling to a new image

Upload the new object, then change `node_image_name`, `node_image_object` and
`node_image_object_version` together. The registration is replaced, which
replaces each node's boot disk and VM. Drain the nodes first, or raise
`node_count` with the new image and lower it afterwards for a blue/green
rollout.

output "node_image" {
  description = "Resolved image reference the pool boots from."
  value       = evroc_custom_disk_image.node.fqid
}

output "image_bucket" {
  description = "Bucket to upload new node images to."
  value       = evroc_bucket.images.name
}

output "node_private_ips" {
  value = evroc_virtual_machine.node[*].private_ipv4_address
}

output "web_server_id" {
  value       = evroc_virtual_machine.web.vm_id
  description = "The unique ID of the web server"
}

output "web_server_status" {
  value       = evroc_virtual_machine.web.status
  description = "The status of the web server"
}

output "public_ip" {
  value       = evroc_public_ip.web.ip_address
  description = "The public IP address of the web server"
}

output "boot_disk_id" {
  value       = evroc_disk.web_boot.disk_id
  description = "The ID of the boot disk"
}

output "data_disk_id" {
  value       = evroc_disk.web_data.disk_id
  description = "The ID of the data disk"
}

output "existing_disk_size" {
  value       = var.lookup_existing_disk ? data.evroc_disk.existing[0].size : null
  description = "Size of the existing disk (if looked up)"
}

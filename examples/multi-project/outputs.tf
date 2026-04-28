# Shared project outputs
output "shared_security_group_name" {
  description = "Name of the canonical security group in the shared project. Dev and prod groups are created with this same name."
  value       = evroc_security_group.shared_web.name
}

output "shared_security_group_id" {
  description = "UUID of the security group in the shared project."
  value       = evroc_security_group.shared_web.sg_id
}

# Dev project outputs
output "dev_security_group_id" {
  description = "UUID of the security group in the dev project."
  value       = evroc_security_group.dev_web.sg_id
}

output "dev_vm_public_ip" {
  description = "Public IP address of the dev VM."
  value       = evroc_virtual_machine.dev_web.public_ipv4_address
}

output "dev_ssh_command" {
  description = "SSH command to connect to the dev VM."
  value       = "ssh evroc-user@${evroc_virtual_machine.dev_web.public_ipv4_address}"
}

# Prod project outputs
output "prod_security_group_id" {
  description = "UUID of the security group in the prod project."
  value       = evroc_security_group.prod_web.sg_id
}

output "prod_vm_public_ip" {
  description = "Public IP address of the prod VM."
  value       = evroc_virtual_machine.prod_web.public_ipv4_address
}

output "prod_ssh_command" {
  description = "SSH command to connect to the prod VM."
  value       = "ssh evroc-user@${evroc_virtual_machine.prod_web.public_ipv4_address}"
}

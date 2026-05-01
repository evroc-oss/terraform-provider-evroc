variable "ssh_public_key" {
  description = "SSH public key for accessing the VMs"
  type        = string
  default     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC... user@example.com"
}

variable "k3s_token" {
  description = "Shared secret for k3s cluster authentication. Must be the same on all nodes."
  type        = string
  sensitive   = true
  default     = "change-me-to-a-random-token"

  validation {
    condition     = length(var.k3s_token) >= 16
    error_message = "k3s_token must be at least 16 characters long for security."
  }
}

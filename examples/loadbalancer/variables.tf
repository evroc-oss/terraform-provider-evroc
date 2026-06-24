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

variable "control_plane_count" {
  description = "Number of control plane nodes (should be odd for etcd quorum: 1, 3, or 5)"
  type        = number
  default     = 3

  validation {
    condition     = var.control_plane_count % 2 == 1
    error_message = "control_plane_count must be odd for etcd quorum."
  }
}

variable "worker_count" {
  description = "Number of worker nodes"
  type        = number
  default     = 3
}

variable "zones" {
  description = "Availability zones to spread nodes across (round-robin). Override with a subset to avoid a degraded zone."
  type        = list(string)
  default     = ["a", "b", "c"]

  validation {
    condition     = length(var.zones) > 0
    error_message = "zones must contain at least one zone."
  }
}

variable "cluster_cidr" {
  description = "Private CIDR the cluster nodes live in; used to scope intra-cluster firewall rules (etcd 2379/2380, kubelet 10250)."
  type        = string
  default     = "10.0.0.0/8"
}

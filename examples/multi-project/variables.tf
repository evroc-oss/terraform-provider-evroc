variable "organization_id" {
  description = "evroc organization ID all projects belong to."
  type        = string
}

variable "shared_project_id" {
  description = "Project ID for the shared networking/security project. This project owns the canonical security group definitions."
  type        = string
}

variable "dev_project_id" {
  description = "Project ID for the development workload project."
  type        = string
}

variable "prod_project_id" {
  description = "Project ID for the production workload project."
  type        = string
}

variable "ssh_public_key" {
  description = "SSH public key to install on all VMs."
  type        = string
}

variable "region" {
  description = "evroc region to deploy into."
  type        = string
  default     = "se-sto"
}

variable "project" {
  description = "evroc project ID"
  type        = string
}

variable "region" {
  description = "evroc region"
  type        = string
  default     = "se-sto"
}

variable "prefix" {
  description = "Prefix for resource names"
  type        = string
  default     = "example"
}

variable "vm_flavor" {
  description = "VM flavor/size"
  type        = string
  default     = "a1a.s"
}

variable "ssh_keys" {
  description = "List of SSH public keys"
  type        = list(string)
  default     = []
}

variable "lookup_existing_disk" {
  description = "Whether to look up an existing disk"
  type        = bool
  default     = false
}

variable "existing_disk_name" {
  description = "Name of existing disk to look up"
  type        = string
  default     = ""
}

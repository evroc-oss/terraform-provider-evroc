variable "node_image_name" {
  type        = string
  description = "Name to register the node image under, e.g. k8s-node-ubuntu2404-v1.35.0. Change it together with the object to roll the pool to a new image."
}

variable "node_image_object" {
  type        = string
  description = "Path of the uploaded image within the bucket, e.g. images/k8s-node-ubuntu2404-v1.35.0.qcow2."
}

variable "node_image_object_version" {
  type        = string
  description = "S3 version ID of the uploaded object. Pins the registration so the image cannot change under its name."
}

variable "pool_name" {
  type        = string
  default     = "workers"
  description = "Prefix for node and disk names."
}

variable "node_count" {
  type    = number
  default = 3
}

variable "node_flavor" {
  type        = string
  default     = "a1a.m"
  description = "Compute profile for the nodes."
}

variable "node_disk_size" {
  type        = number
  default     = 50
  description = "Default and boot disk size in GB. At least the image's virtual disk size."
}

variable "zone" {
  type    = string
  default = "a"
}

variable "ssh_public_key" {
  type = string
}

variable "security_group_fqids" {
  type        = list(string)
  default     = []
  description = "Security groups to attach to the nodes."
}

variable "kubeadm_join_command" {
  type        = string
  sensitive   = true
  description = "Output of 'kubeadm token create --print-join-command' on the control plane."
}

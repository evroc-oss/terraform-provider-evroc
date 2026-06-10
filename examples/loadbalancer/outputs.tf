output "kubernetes_api_endpoint" {
  description = "Kubernetes API endpoint (via load balancer)"
  value       = "https://${evroc_public_ip.k8s_api.ip_address}:6443"
}

output "lb_fqid" {
  description = "Fully qualified ID of the API load balancer"
  value       = evroc_loadbalancer.k8s_api.fqid
}

output "lb_pool_name" {
  description = "Backend pool name"
  value       = evroc_lb_backend_pool.k8s_api.name
}

output "registration_address" {
  description = "Private address that secondary control planes and workers join directly during bootstrap"
  value       = "https://${evroc_virtual_machine.control_plane_first.private_ipv4_address}:6443"
}

output "control_plane_nodes" {
  description = "Control plane node names and zones"
  value = concat(
    [{ name = evroc_virtual_machine.control_plane_first.name, zone = evroc_disk.control_plane_first.zone }],
    [for i, vm in evroc_virtual_machine.control_plane_rest : { name = vm.name, zone = evroc_disk.control_plane_rest[i].zone }],
  )
}

output "worker_nodes" {
  description = "Worker node names and zones"
  value = [for i, vm in evroc_virtual_machine.worker : {
    name = vm.name
    zone = evroc_disk.worker[i].zone
  }]
}

output "kubeconfig_command" {
  description = "Command to fetch kubeconfig from the first control plane node"
  value       = "scp evroc-user@<cp-1-ip>:/etc/rancher/k3s/k3s.yaml kubeconfig.yaml && sed -i 's/127.0.0.1/${evroc_public_ip.k8s_api.ip_address}/g' kubeconfig.yaml"
}

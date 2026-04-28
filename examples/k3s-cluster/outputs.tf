output "control_plane_public_ip" {
  description = "Public IP address of the k3s control plane node"
  value       = evroc_public_ip.control_plane.ip_address
}

output "worker1_public_ip" {
  description = "Public IP address of worker node 1"
  value       = evroc_public_ip.worker1.ip_address
}

output "worker2_public_ip" {
  description = "Public IP address of worker node 2"
  value       = evroc_public_ip.worker2.ip_address
}

output "kubernetes_api_endpoint" {
  description = "Kubernetes API endpoint"
  value       = "https://${evroc_public_ip.control_plane.ip_address}:6443"
}

output "ssh_control_plane" {
  description = "SSH command for control plane node"
  value       = "ssh ubuntu@${evroc_public_ip.control_plane.ip_address}"
}

output "ssh_worker1" {
  description = "SSH command for worker node 1"
  value       = "ssh ubuntu@${evroc_public_ip.worker1.ip_address}"
}

output "ssh_worker2" {
  description = "SSH command for worker node 2"
  value       = "ssh ubuntu@${evroc_public_ip.worker2.ip_address}"
}

output "kubeconfig_command" {
  description = "Command to fetch kubeconfig from control plane"
  value       = "ssh ubuntu@${evroc_public_ip.control_plane.ip_address} 'sudo cat /etc/rancher/k3s/k3s.yaml' | sed 's/127.0.0.1/${evroc_public_ip.control_plane.ip_address}/g' > kubeconfig.yaml"
}

output "cluster_info" {
  description = "k3s cluster information"
  value = {
    control_plane_ip = evroc_public_ip.control_plane.ip_address
    worker_ips = [
      evroc_public_ip.worker1.ip_address,
      evroc_public_ip.worker2.ip_address
    ]
    api_endpoint      = "https://${evroc_public_ip.control_plane.ip_address}:6443"
    security_group_id = evroc_security_group.k3s.sg_id
  }
}

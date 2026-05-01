# k3s Kubernetes Cluster on evroc

This example demonstrates how to deploy a complete **3-node k3s Kubernetes cluster** on evroc using Terraform.

## Architecture

This configuration creates:

- **1 Control Plane Node** - Runs the k3s server with Kubernetes API, scheduler, and controller manager
- **2 Worker Nodes** - Run k3s agents for hosting application workloads
- **Security Group** - Firewall rules for k3s cluster communication
- **Public IPs** - External access for all nodes and Kubernetes API

### Cluster Topology

```
┌─────────────────────────────────────────────────────────┐
│                    k3s Cluster                          │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────────┐                                  │
│  │  Control Plane   │  (k3s server)                    │
│  │   Public IP      │  - Kubernetes API: 6443          │
│  │   a1a.s / 50GB   │  - k3s supervisor: 9345          │
│  └──────────────────┘                                  │
│           │                                             │
│           │ k3s cluster network                         │
│           │                                             │
│  ┌────────┴─────────┐                                  │
│  │                  │                                   │
│  v                  v                                   │
│  ┌────────┐    ┌────────┐                              │
│  │Worker 1│    │Worker 2│  (k3s agents)                │
│  │a1a.s   │    │a1a.s   │                              │
│  │50GB    │    │50GB    │                              │
│  └────────┘    └────────┘                              │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Features

- ✅ **Fully automated deployment** - From VMs to running k3s cluster
- ✅ **High availability ready** - Can be extended to multi-control-plane
- ✅ **Cloud-init provisioning** - Automated k3s installation
- ✅ **Security groups** - Proper firewall configuration
- ✅ **Topology labels** - Zone awareness for workload placement
- ✅ **Kubeconfig access** - Easy cluster access configuration

## Prerequisites

1. **evroc Account** with API credentials
2. **Terraform** >= 1.0
3. **SSH Key Pair** for VM access
4. **kubectl** (optional, for cluster access)

## Configuration

### Required Variables

Create a `terraform.tfvars` file:

```hcl
ssh_public_key = "ssh-rsa AAAAB3NzaC... your-email@example.com"
k3s_token      = "my-super-secret-k3s-token-min-16-chars"
```

### Optional Variables

```hcl
k3s_version = "v1.28.5+k3s1"  # Specific k3s version (default: latest stable)
```

## Deployment

### Step 1: Initialize Terraform

```bash
terraform init
```

### Step 2: Review the Plan

```bash
terraform plan
```

This will create:
- 3 virtual machines (1 control plane + 2 workers)
- 3 disks (boot disks for each VM)
- 3 public IPs
- 1 security group with k3s firewall rules

### Step 3: Deploy the Cluster

```bash
terraform apply
```

**Deployment time:** Approximately 3-5 minutes for full cluster bootstrap.

### Step 4: Verify Deployment

After `terraform apply` completes, you'll see outputs with cluster information:

```bash
Outputs:

control_plane_public_ip = "203.0.113.10"
kubernetes_api_endpoint = "https://203.0.113.10:6443"
ssh_control_plane = "ssh ubuntu@203.0.113.10"
worker1_public_ip = "203.0.113.11"
worker2_public_ip = "203.0.113.12"
kubeconfig_command = "ssh ubuntu@203.0.113.10 'sudo cat /etc/rancher/k3s/k3s.yaml' | sed 's/127.0.0.1/203.0.113.10/g' > kubeconfig.yaml"
```

## Accessing the Cluster

### Option 1: Fetch Kubeconfig

Use the provided command from outputs:

```bash
# Fetch kubeconfig from control plane
ssh ubuntu@<CONTROL_PLANE_IP> 'sudo cat /etc/rancher/k3s/k3s.yaml' | \
  sed 's/127.0.0.1/<CONTROL_PLANE_IP>/g' > kubeconfig.yaml

# Set KUBECONFIG environment variable
export KUBECONFIG=$(pwd)/kubeconfig.yaml

# Verify cluster access
kubectl get nodes
```

Expected output:
```
NAME                STATUS   ROLES                  AGE   VERSION
k3s-control-plane   Ready    control-plane,master   5m    v1.28.5+k3s1
k3s-worker1         Ready    worker                 4m    v1.28.5+k3s1
k3s-worker2         Ready    worker                 4m    v1.28.5+k3s1
```

### Option 2: SSH to Control Plane

```bash
# SSH to control plane node
ssh ubuntu@<CONTROL_PLANE_IP>

# Use kubectl directly (k3s installs it)
kubectl get nodes
kubectl get pods -A
```

## Security Group Rules

The cluster security group includes:

| Port  | Protocol | Direction | Purpose                    |
|-------|----------|-----------|----------------------------|
| 22    | TCP      | Ingress   | SSH access                 |
| 6443  | TCP      | Ingress   | Kubernetes API server      |
| 9345  | TCP      | Ingress   | k3s supervisor (for agents)|
| 80    | TCP      | Ingress   | HTTP ingress              |
| 443   | TCP      | Ingress   | HTTPS ingress             |
| All   | All      | Egress    | Outbound traffic          |

## k3s Configuration

### Control Plane (Server)

Configuration in `/etc/rancher/k3s/config.yaml`:

```yaml
write-kubeconfig-mode: "0644"
tls-san:
  - "<PUBLIC_IP>"
  - "k3s-control-plane"
disable:
  - traefik      # Disabled to allow custom ingress
  - servicelb    # Disabled to allow custom load balancer
  - local-storage # Disabled to use evroc CSI driver
node-label:
  - "node-role.kubernetes.io/control-plane=true"
  - "topology.kubernetes.io/zone=se-sto"
```

### Workers (Agents)

Configuration in `/etc/rancher/k3s/config.yaml`:

```yaml
server: "https://<CONTROL_PLANE_IP>:6443"
node-label:
  - "node-role.kubernetes.io/worker=true"
  - "topology.kubernetes.io/zone=se-sto"
```

## Common Operations

### Deploy a Test Application

```bash
# Create a simple nginx deployment
kubectl create deployment nginx --image=nginx --replicas=2

# Expose it as a NodePort service
kubectl expose deployment nginx --port=80 --type=NodePort

# Get the NodePort
kubectl get svc nginx
```

Access via: `http://<WORKER_IP>:<NODEPORT>`

### Check Cluster Status

```bash
# Node status
kubectl get nodes -o wide

# All pods across namespaces
kubectl get pods -A

# k3s system components
kubectl get pods -n kube-system
```

### View Node Labels

```bash
# Check topology zone labels
kubectl get nodes --show-labels | grep topology

# Check role labels
kubectl get nodes --show-labels | grep node-role
```

## Customization

### Change VM Flavor

Edit `main.tf` and change the `flavor` parameter:

```hcl
resource "evroc_virtual_machine" "control_plane" {
  name   = "k3s-control-plane"
  flavor = "a1a.m"  # Upgrade to medium flavor
  # ...
}
```

### Add More Workers

Copy the `worker2` resource block and adjust the name:

```hcl
resource "evroc_virtual_machine" "worker3" {
  name      = "k3s-worker3"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.worker3_boot.fqid
  # ... (same pattern as worker1/worker2)
}
```

### Install evroc CSI Driver

After cluster is running:

```bash
# Clone the CSI driver
git clone https://github.com/evroc-oss/evroc-csi-driver.git
cd evroc-csi-driver

# Install with Helm
helm install evroc-csi ./charts/evroc-csi-driver \
  --namespace kube-system \
  --set storageClass.default=true
```

## Troubleshooting

### Check Cloud-Init Progress

```bash
# SSH to a node
ssh ubuntu@<NODE_IP>

# View cloud-init logs
sudo tail -f /var/log/cloud-init-output.log

# Check cloud-init status
sudo cloud-init status
```

### Check k3s Service Status

```bash
# Control plane
sudo systemctl status k3s

# Workers
sudo systemctl status k3s-agent
```

### View k3s Logs

```bash
# Control plane logs
sudo journalctl -u k3s -f

# Worker logs
sudo journalctl -u k3s-agent -f
```

### Worker Not Joining

If a worker doesn't join the cluster:

1. **Check token** - Ensure `k3s_token` is the same on all nodes
2. **Check network** - Verify security group allows port 6443 and 9345
3. **Check timing** - Workers wait 30s for control plane, may need longer

```bash
# On worker, check if it can reach control plane
curl -k https://<CONTROL_PLANE_IP>:6443
```

## Cleanup

To destroy all resources:

```bash
terraform destroy
```

This will remove:
- All 3 VMs
- All disks
- All public IPs
- The security group

**Warning:** This is permanent. Ensure you've backed up any data before destroying.

## Cost Estimation

Based on typical evroc pricing (check current rates):

| Resource Type | Quantity | Monthly Cost (approx) |
|---------------|----------|-----------------------|
| a1a.s VMs     | 3        | ~$90-150              |
| 50GB Disks    | 3        | ~$15-30               |
| Public IPs    | 3        | ~$5-10                |
| **Total**     |          | **~$110-190/month**   |

## Next Steps

1. **Install CNI plugin** (if not using default flannel)
2. **Install evroc CSI driver** for persistent storage
3. **Set up ingress controller** (nginx, traefik, etc.)
4. **Configure cert-manager** for TLS certificates
5. **Deploy your applications**

## Related Examples

- [**networking/**](../networking/) - Security groups and networking setup
- [**disk-attachment/**](../disk-attachment/) - Hot-attach persistent storage
- [**storage/**](../storage/) - S3-compatible object storage

## Resources

- [k3s Documentation](https://docs.k3s.io/)
- [evroc Documentation](https://docs.cloud.evroc.com)
- [evroc CSI Driver](https://github.com/evroc-oss/evroc-csi-driver)
- [Terraform evroc Provider](https://github.com/evroc-oss/terraform-provider-evroc)

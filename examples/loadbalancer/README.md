<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- SPDX-FileCopyrightText: 2026 evroc -->

# HA k3s cluster behind an evroc load balancer

Provisions a highly-available [k3s](https://k3s.io) Kubernetes cluster on evroc:
a 3-node embedded-etcd control plane plus workers, spread across availability
zones, with an evroc **load balancer as the public Kubernetes API endpoint**.

## Architecture

- **The first control plane** founds the cluster (`--cluster-init`).
- **Secondary control planes and workers join an explicit member** (the first
  control plane's private IP), which is the standard approach for clustered
  systems. Once joined, k3s discovers every apiserver and load-balances across
  them client-side, so nodes are not pinned to the founder at runtime.
- **The load balancer is the public API endpoint** for external clients
  (`kubectl`). The control planes' served certs include the LB address
  (`--tls-san`) so TLS validates.

After the cluster has formed, all control planes are equal — k3s discovers every
apiserver and etcd maintains quorum across any 2 of 3 nodes, so losing the
founder has no impact.

### Day-2: adding nodes after the cluster is running

The cloud-init templates use the first control plane's private IP as the join
address. If CP-1 is unavailable when you need to scale up or replace a node,
point `K3S_URL` at any other living control plane's IP instead.

## Required firewall ports

An HA control plane needs more than the API port. This example's security group
opens:

| Port | Purpose | Scope |
|------|---------|-------|
| 6443 | Kubernetes API | public |
| 9345 | k3s supervisor / registration | public |
| 2379 | embedded etcd client | cluster CIDR |
| 2380 | embedded etcd **peer** (required between control planes) | cluster CIDR |
| 10250 | kubelet API (`kubectl logs/exec`, metrics-server) | cluster CIDR |
| 22 | SSH | public |

Without **2379/2380 between control planes the HA cluster cannot form**.

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars   # set ssh_public_key + a >=16-char k3s_token
terraform plan
terraform apply

# endpoint + how to get a kubeconfig:
terraform output kubernetes_api_endpoint
terraform output kubeconfig_command

terraform destroy
```

### Notable variables

| Variable | Default | Notes |
|----------|---------|-------|
| `control_plane_count` | `3` | Odd for etcd quorum (1/3/5). |
| `worker_count` | `3` | |
| `zones` | `["a","b","c"]` | Round-robin placement. Set a subset (e.g. `["a","b"]`) to avoid a degraded zone. |
| `cluster_cidr` | `10.0.0.0/8` | Scopes the intra-cluster etcd/kubelet rules. |

# HTTPS load balancer

Terminate HTTPS at a load balancer and forward HTTP requests to existing VMs on port 8080.
The VMs must serve `/healthz`, be in the configured project and region, and use the default VPC.
For a custom VPC, add a `backend_network` block to the load balancer; see the
[load balancer resource](../../docs/resources/loadbalancer.md).

Configure evroc authentication as described in the [provider setup](../../README.md),
then supply `project`, `region`, `backend_refs`, `hostname`, `certificate_file`, and
`private_key_file` through Terraform variables. The certificate must cover `hostname`.
The chain must contain the leaf certificate followed by intermediates, without the root CA.

Certificate and key contents are marked sensitive but are stored in Terraform state.
Protect the state and saved plans. Certificate issuance and renewal scheduling remain
your responsibility. Replace either PEM file and apply to upload the complete pair to
the same certificate resource; listeners keep their reference.

```shell
terraform init
terraform plan -out=tfplan
terraform apply tfplan
```

Use the public IPv4 output to verify a request before configuring DNS. Replace the
hostname and address below with your values:

```shell
curl --resolve web.example.com:443:PUBLIC_IPV4 https://web.example.com/
```

Check the response and your application's access logs. For a private CA, add
`--cacert` with the CA certificate file. Configure DNS to point your hostname at the
public IP once the application responds as expected.

The route's rule has no `match` blocks, so it accepts all requests for the configured
hostname. See the [HTTP route example](../resources/evroc_lb_http_route/resource.tf)
for path, method, header, and query parameter matching. HTTP listeners can reference
the same route on port 80 without a `tls` block; they do not redirect to HTTPS.

When finished, remove the resources with `terraform destroy`.

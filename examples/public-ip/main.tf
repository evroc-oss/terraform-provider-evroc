terraform {
  required_providers {
    evroc = {
      source  = "github.com/evroc-oss/evroc"
      version = "~> 0.1"
    }
  }
}

provider "evroc" {
  # Credentials and context read from ~/.evroc/config.yaml by default.
  # See: https://github.com/evroc-oss/terraform-provider-evroc#1-authenticate-with-evroc
}

# Allocate a public IP
resource "evroc_public_ip" "example" {
  name = "example-public-ip"
}

# Output the IP address
output "ip_address" {
  value       = evroc_public_ip.example.ip_address
  description = "The allocated public IPv4 address"
}

output "ip_id" {
  value       = evroc_public_ip.example.ip_id
  description = "The unique ID of the public IP"
}

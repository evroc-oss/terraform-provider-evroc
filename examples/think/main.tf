terraform {
  required_providers {
    evroc = {
      source = "github.com/evroc-oss/evroc"
    }
  }
}

provider "evroc" {
  # Credentials and context read from ~/.evroc/config.yaml by default.
  # See: https://github.com/evroc-oss/terraform-provider-evroc#1-authenticate-with-evroc
}

# List available models and sizes
data "evroc_think_models" "available" {}
data "evroc_think_sizes" "available" {}

# Create a dedicated inference instance
resource "evroc_think_instance" "llama" {
  name  = "my-llama-instance"
  model = "meta-llama/Llama-3.3-70B-Instruct"
  # size = "a100.2x" # Optional: override the default size for the model
}

# Create an API key to authenticate requests
resource "evroc_think_api_key" "app" {
  name   = "my-app-key"
  expiry = "2027-01-01T00:00:00Z" # Optional: omit for a non-expiring key
}

# Query instance details
data "evroc_think_instance" "llama" {
  name = evroc_think_instance.llama.name
}

# Outputs
output "available_models" {
  description = "List of available Think models"
  value       = data.evroc_think_models.available.models[*].name
}

output "available_sizes" {
  description = "List of available Think GPU sizes"
  value       = data.evroc_think_sizes.available.sizes[*].name
}

output "instance_endpoint" {
  description = "OpenAI-compatible API endpoint for the instance"
  value       = evroc_think_instance.llama.endpoint
}

output "instance_phase" {
  description = "Current instance lifecycle phase"
  value       = evroc_think_instance.llama.phase
}

output "api_key_token" {
  description = "API key secret. The API only returns this at creation; Terraform persists it in state."
  value       = evroc_think_api_key.app.token
  sensitive   = true
}

output "api_key_prefix" {
  description = "API key prefix for identification"
  value       = evroc_think_api_key.app.token_prefix
}

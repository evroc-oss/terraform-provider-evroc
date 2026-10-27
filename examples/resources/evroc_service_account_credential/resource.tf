resource "evroc_service_account" "application" {
  name    = "application"
  enabled = true
}

# Set this to the intended expiration time. Updating it replaces the credential.
variable "credential_expires_at" {
  type        = string
  description = "Credential expiration in RFC3339 format."
}

resource "evroc_service_account_credential" "hmac" {
  name                = "application-hmac"
  service_account_ref = evroc_service_account.application.fqid
  type                = "hmac-sigv4"
  expires_at          = var.credential_expires_at
}

# Assign the service account appropriate roles separately.
output "access_key_id" {
  value = evroc_service_account_credential.hmac.access_key_id
}

output "secret_access_key" {
  value     = evroc_service_account_credential.hmac.secret_access_key
  sensitive = true
}

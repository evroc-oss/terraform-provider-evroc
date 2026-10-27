data "evroc_service_account_credential" "existing" {
  name                = "application-hmac"
  project             = "my-project"
  service_account_ref = "/iam/projects/my-project/serviceAccounts/application"
}

# Only metadata is available; this lookup cannot recover the secret.
output "access_key_id" {
  value = data.evroc_service_account_credential.existing.access_key_id
}

resource "evroc_kms_key" "application" {
  name                   = "application-key"
  auto_rotate_after_days = 30
  expire_after_days      = 365
  enabled                = true

  lifecycle {
    prevent_destroy = true
  }
}

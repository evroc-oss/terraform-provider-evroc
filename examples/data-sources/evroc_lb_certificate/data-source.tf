data "evroc_lb_certificate" "web" {
  name = "web-cert"
}

output "certificate_expiry" {
  value = data.evroc_lb_certificate.web.not_after
}

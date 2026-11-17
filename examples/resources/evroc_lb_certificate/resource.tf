resource "evroc_lb_certificate" "web" {
  name            = "web-cert"
  certificate_pem = file("fullchain.pem")
  private_key_pem = file("privkey.pem")
}

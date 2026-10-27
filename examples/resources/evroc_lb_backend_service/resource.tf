resource "evroc_lb_backend_pool" "application" {
  name = "application-pool"
  # Add backend_refs for the VMs serving the application.
}

resource "evroc_lb_backend_service" "application" {
  name             = "application-service"
  port             = 8080
  backend_pool_ref = evroc_lb_backend_pool.application.fqid
  proxy_protocol   = true

  health_check {
    target_port            = 8081
    disable_proxy_protocol = true
    http {
      path = "/healthz"
    }
  }
}

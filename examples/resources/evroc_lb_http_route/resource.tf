resource "evroc_lb_http_route" "web" {
  name      = "web-route"
  hostnames = ["web.example.com"]

  rule {
    name                = "api"
    backend_service_ref = evroc_lb_backend_service.api.fqid

    match {
      method = "GET"
      path {
        type  = "PathPrefix"
        value = "/api"
      }
      header {
        name  = "x-version"
        value = "v2"
      }
      query_param {
        name  = "preview"
        value = "true"
      }
    }
  }

  rule {
    name                = "fallback"
    backend_service_ref = evroc_lb_backend_service.web.fqid
  }
}

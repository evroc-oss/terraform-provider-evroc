data "evroc_virtual_machine_console_logs" "debug" {
  name = "application-vm"
}

output "console_logs" {
  value     = data.evroc_virtual_machine_console_logs.debug.logs
  sensitive = true
}

data "mijnhost_domain" "example" {
  domain = "example.com"
}

output "renewal_date" {
  value = data.mijnhost_domain.example.renewal_date
}

output "status" {
  value = data.mijnhost_domain.example.status
}

output "tags" {
  value = data.mijnhost_domain.example.tags
}

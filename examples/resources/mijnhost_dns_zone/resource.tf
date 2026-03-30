# Manage the complete DNS zone for a domain.
# WARNING: Any records not listed here will be DELETED from the zone.
# This resource does a full replace on every apply.

resource "mijnhost_dns_zone" "example" {
  domain = "example.com"

  records = [
    {
      type  = "A"
      name  = "example.com"
      value = "1.2.3.4"
      ttl   = 3600
    },
    {
      type  = "A"
      name  = "www.example.com"
      value = "1.2.3.4"
      ttl   = 3600
    },
    {
      type  = "MX"
      name  = "example.com"
      value = "10 mail.example.com"
      ttl   = 3600
    },
    {
      type  = "TXT"
      name  = "example.com"
      value = "v=spf1 include:spf.example.com ~all"
      ttl   = 3600
    },
  ]
}

# Manage individual DNS records.
# Changes to domain, type, name, or value force a new resource.
# Only TTL can be updated in place.

resource "mijnhost_dns_record" "www" {
  domain = "example.com"
  type   = "A"
  name   = "www.example.com"
  value  = "1.2.3.4"
  ttl    = 3600
}

resource "mijnhost_dns_record" "apex" {
  domain = "example.com"
  type   = "A"
  name   = "example.com"
  value  = "1.2.3.4"
  ttl    = 3600
}

resource "mijnhost_dns_record" "mail" {
  domain = "example.com"
  type   = "MX"
  name   = "example.com"
  value  = "10 mail.example.com"
  ttl    = 3600
}

resource "mijnhost_dns_record" "spf" {
  domain = "example.com"
  type   = "TXT"
  name   = "example.com"
  value  = "v=spf1 include:spf.example.com ~all"
  ttl    = 3600
}

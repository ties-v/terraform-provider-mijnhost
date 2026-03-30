# WARNING: mijnhost_dns_zone replaces the ENTIRE record set on every apply.
# Any record not listed here — including records added outside of Terraform
# or created automatically by mijn.host — will be permanently deleted.
#
# Before applying this to an existing domain, check the current records:
#   dig +short NS example.com
#   dig +short ANY example.com
#
# Use mijnhost_dns_record instead if you only want to manage specific records
# without affecting records maintained outside of Terraform.

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

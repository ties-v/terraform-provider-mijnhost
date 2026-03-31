# terraform-provider-mijnhost

An OpenTofu / Terraform provider for managing DNS records on [mijn.host](https://mijn.host).

## Requirements

- [OpenTofu](https://opentofu.org) >= 1.6 or [Terraform](https://www.terraform.io) >= 1.5
- A mijn.host account with API access enabled

## Installation

Add the provider to your `required_providers` block and run `tofu init`:

```hcl
terraform {
  required_providers {
    mijnhost = {
      source  = "ties-v/mijnhost"
      version = "~> 0.1"
    }
  }
}
```

## Getting an API key

Log in to the [mijn.host control panel](https://mijn.host/cp/), navigate to **Account → API Access**, and follow the instructions to generate an API key.

## Building from source

Requires [Go](https://golang.org) >= 1.21.

```bash
git clone https://github.com/ties-v/terraform-provider-mijnhost.git
cd terraform-provider-mijnhost
make install
```

This places the binary in `~/.terraform.d/plugins/registry.terraform.io/ties-v/mijnhost/0.1.0/linux_amd64/`.

## Usage

### Provider configuration

```hcl
terraform {
  required_providers {
    mijnhost = {
      source  = "ties-v/mijnhost"
      version = "~> 0.1"
    }
  }
}

provider "mijnhost" {
  # api_key = "your-api-key"
  # Alternatively, set the MIJNHOST_API_KEY environment variable.
}
```

The API key can be provided via:
- The `api_key` provider argument (mark it sensitive or use a variable)
- The `MIJNHOST_API_KEY` environment variable (recommended)

---

## Resources

### `mijnhost_dns_record`

Manages a **single DNS record** for a domain. Other records in the zone are not affected.

Because the mijn.host API has no per-record create or delete endpoint, this resource uses a read-modify-write strategy: it reads the full record set, adds or removes the target record, then writes the full set back. A per-domain mutex ensures that multiple `mijnhost_dns_record` resources on the same domain are applied sequentially within a single Terraform run.

Changes to `domain`, `type`, `name`, or `value` force a new resource. Only `ttl` can be updated in place.

#### Example

```hcl
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
```

#### Argument reference

| Argument | Type   | Required | Description |
|----------|--------|----------|-------------|
| `domain` | string | yes | The domain name (e.g. `example.com`). Forces new resource on change. |
| `type`   | string | yes | DNS record type (`A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `SRV`, `CAA`, …). Forces new resource on change. |
| `name`   | string | yes | Record hostname (e.g. `www.example.com` or `example.com` for apex). Forces new resource on change. |
| `value`  | string | yes | Record value (e.g. an IP address, mail exchanger, or quoted TXT string). Forces new resource on change. |
| `ttl`    | number | yes | Time to live in seconds. Can be updated in place. |

#### Attribute reference

| Attribute | Description |
|-----------|-------------|
| `id` | Composite identifier: `domain/type/name/value`. |

---

### `mijnhost_dns_zone`

Manages the **complete set of DNS records** for a domain. On every apply, the provider replaces the entire record set with exactly what is listed in this resource. Any record not listed — including records added outside of Terraform, records created by mijn.host automatically (e.g. default NS records), or records managed by other tools — **will be permanently deleted**.

> **Warning:** Before using this resource on an existing domain, retrieve the current records (e.g. via `dig` or the mijn.host control panel) and include all records you want to keep in the `records` set.

Use this resource when you want full declarative control over a zone and accept that Terraform is the sole source of truth. Use `mijnhost_dns_record` instead when:
- records are managed outside Terraform (e.g. by mijn.host automatically), or
- you only want to manage a subset of records without affecting others.

Deleting this resource removes it from Terraform state only — the DNS records remain in mijn.host, because the API does not accept an empty record set.

#### Example

```hcl
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
```

#### Argument reference

| Argument  | Type   | Required | Description |
|-----------|--------|----------|-------------|
| `domain`  | string | yes | The domain name. Forces new resource on change. |
| `records` | set    | yes | The complete desired set of DNS records (see nested schema below). |

**`records` nested schema:**

| Argument | Type   | Required | Description |
|----------|--------|----------|-------------|
| `type`   | string | yes | DNS record type. |
| `name`   | string | yes | Record hostname. |
| `value`  | string | yes | Record value. |
| `ttl`    | number | yes | Time to live in seconds. |

#### Attribute reference

| Attribute | Description |
|-----------|-------------|
| `id` | The domain name. |

---

## Data sources

### `mijnhost_domain`

Retrieves information about a domain in your mijn.host account.

#### Example

```hcl
data "mijnhost_domain" "example" {
  domain = "example.com"
}

output "renewal_date" {
  value = data.mijnhost_domain.example.renewal_date
}
```

#### Argument reference

| Argument | Type   | Required | Description |
|----------|--------|----------|-------------|
| `domain` | string | yes | The domain name to look up. |

#### Attribute reference

| Attribute      | Description |
|----------------|-------------|
| `id`           | The domain name. |
| `renewal_date` | Renewal date in `YYYY-MM-DD` format. |
| `status`       | Domain status (e.g. `active`, `Cancelled`). |
| `tags`         | List of tags associated with the domain. |

---

## API notes

- The mijn.host API endpoint is `https://mijn.host/api/v2`.
- DNS management uses a full-replace model (`PUT /domains/{domain}/dns`). The provider handles the necessary read-modify-write for individual record management.
- Record names must include a trailing dot when sent to the API (`example.com.`). The provider handles this normalization automatically — always specify names **without** a trailing dot in your Terraform configuration.

## License

MIT

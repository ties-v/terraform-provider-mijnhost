# CLAUDE.md

## Project overview

OpenTofu / Terraform provider for managing DNS records on mijn.host. Written in Go using the Terraform Plugin Framework (v2+, protocol v6).

## Common commands

```bash
make build      # compile the provider binary
make install    # build + install to ~/.terraform.d/plugins/
make test       # unit tests (no API key needed)
make testacc    # acceptance tests (requires env vars — see below)
make fmt        # gofmt -s -w .
make vet        # go vet ./...
make docs       # regenerate docs/ from schema + examples/ using tfplugindocs
```

Go is installed at `/var/lib/snapd/snap/bin/go` on this machine.

## Architecture

```
main.go                              # providerserver entry point
internal/
  mijnhost/
    client.go                        # HTTP client for the mijn.host API
    client_test.go                   # unit tests (httptest mock server)
  provider/
    provider.go                      # provider config (api_key / MIJNHOST_API_KEY)
    dns_record_resource.go           # mijnhost_dns_record — individual record management
    dns_zone_resource.go             # mijnhost_dns_zone — full zone replacement
    domain_data_source.go            # mijnhost_domain — read-only domain info
    helpers_test.go                  # unit tests for unexported helpers (same package)
    *_test.go                        # acceptance tests (provider_test package)
```

## API constraints

- Base URL: `https://mijn.host/api/v2`
- Auth: `API-Key` request header
- **No per-record create or delete endpoint exists.** DNS changes use:
  - `GET /domains/{domain}/dns` — read full record set
  - `PUT /domains/{domain}/dns` — replace the entire record set
  - `PATCH /domains/{domain}/dns` — update a single record (matched by type+name)
- The API requires record names with a trailing dot (`example.com.`) in request bodies, but returns them without in GET responses. `NormalizeName` / `APIName` in `client.go` handle this.

## Key design decisions

- `mijnhost_dns_record` uses **read-modify-write** (GET + filter/append + PUT) for create and delete; PATCH for TTL-only updates. `type`, `name`, `value`, and `domain` are all `RequiresReplace`.
- `mijnhost_dns_zone` Delete is a **no-op** — the API rejects empty record sets, so destroying the resource only removes it from state.
- Record ID format: `domain/type/name/value` (value is last and may contain slashes — parsed with `strings.SplitN(..., 4)`).
- Names are stored in state **without** trailing dots. The client adds/removes them transparently.

## Testing

### Unit tests
Run without any credentials:
```bash
make test
```

### Acceptance tests
Require a real mijn.host account. Tests skip automatically if env vars are missing.

```bash
export MIJNHOST_API_KEY=your-key
export MIJNHOST_TEST_DOMAIN=example.com          # records will be added/removed
export MIJNHOST_TEST_ZONE_DOMAIN=zone-test.com   # WARNING: ALL records overwritten
make testacc
```

`MIJNHOST_TEST_ZONE_DOMAIN` gates the `dns_zone` acceptance test. Use a domain whose records can be safely wiped.

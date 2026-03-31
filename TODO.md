# TODO

## Publish the provider

- [x] Set up GPG key for release signing
- [x] Create a GoReleaser config to build and sign binaries for all target platforms
- [x] Create a GitHub Actions workflow that triggers a release on tag push
- [x] Register the provider on the OpenTofu registry — PR pending merge
- [x] Update README with installation instructions once the registry PR is merged

## Test against a real domain

- [x] Run `make testacc` with `MIJNHOST_API_KEY` and `MIJNHOST_TEST_DOMAIN` set
- [x] Set `MIJNHOST_TEST_ZONE_DOMAIN` and run the zone resource acceptance test (currently always skipped)
- [x] Confirm all idempotency steps pass (re-apply produces an empty plan)

## Handle operational edge cases

- [x] **Concurrent applies** — `mijnhost_dns_record` uses read-modify-write; two resources on the same domain applied in parallel can race and lose records. Fixed with a per-domain mutex on the shared client (`Client.LockDomain`). Note: only protects within a single Terraform run; concurrent separate processes are not protected.
- [x] **Records outside Terraform** — document clearly that `mijnhost_dns_zone` will delete any record not listed in the resource, including records created outside Terraform (e.g. auto-added by mijn.host).
- ~~**API key rotation**~~ — out of scope.

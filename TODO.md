# TODO

## Publish the provider

- [ ] Set up GPG key for release signing
- [ ] Create a GoReleaser config to build and sign binaries for all target platforms
- [ ] Create a GitHub Actions workflow that triggers a release on tag push
- [ ] Register the provider on the OpenTofu registry and/or Terraform registry
- [ ] Update README with installation instructions once published (remove "build from source" as the primary path)

## Test against a real domain

- [x] Run `make testacc` with `MIJNHOST_API_KEY` and `MIJNHOST_TEST_DOMAIN` set
- [x] Set `MIJNHOST_TEST_ZONE_DOMAIN` and run the zone resource acceptance test (currently always skipped)
- [x] Confirm all idempotency steps pass (re-apply produces an empty plan)

## Handle operational edge cases

- [x] **Concurrent applies** — `mijnhost_dns_record` uses read-modify-write; two resources on the same domain applied in parallel can race and lose records. Fixed with a per-domain mutex on the shared client (`Client.LockDomain`). Note: only protects within a single Terraform run; concurrent separate processes are not protected.
- [x] **Records outside Terraform** — document clearly that `mijnhost_dns_zone` will delete any record not listed in the resource, including records created outside Terraform (e.g. auto-added by mijn.host).
- [ ] **API key rotation** — document the process for updating `MIJNHOST_API_KEY` without causing a provider outage.

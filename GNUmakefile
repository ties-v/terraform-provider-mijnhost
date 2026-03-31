default: build

build:
	go build -o terraform-provider-mijnhost .

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/ties-v/mijnhost/0.1.0/linux_amd64
	cp terraform-provider-mijnhost ~/.terraform.d/plugins/registry.terraform.io/ties-v/mijnhost/0.1.0/linux_amd64/

# Unit tests — no real API required.
test:
	go test ./... -v -skip "^TestAcc"

# Acceptance tests — require a real mijn.host account.
#
# Required environment variables:
#   MIJNHOST_API_KEY         — your mijn.host API key
#   MIJNHOST_TEST_DOMAIN     — a domain you own (records will be added/removed)
#
# Optional:
#   MIJNHOST_TEST_ZONE_DOMAIN — a separate domain safe for full-zone replacement
#                               (WARNING: all records will be overwritten)
#
testacc:
	TF_ACC=1 go test ./internal/provider/... -v -timeout 120s

fmt:
	gofmt -s -w .

vet:
	go vet ./...

docs:
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	tfplugindocs generate --provider-name mijnhost

.PHONY: build install test testacc fmt vet docs

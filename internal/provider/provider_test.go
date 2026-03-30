package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ties-v/terraform-provider-mijnhost/internal/provider"
)

// testAccProtoV6ProviderFactories is used in acceptance tests to create a
// provider server backed by the real provider implementation.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"mijnhost": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccPreCheck validates that the required environment variables for
// acceptance tests are set.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("MIJNHOST_API_KEY"); v == "" {
		t.Skip("MIJNHOST_API_KEY must be set to run acceptance tests")
	}
	if v := os.Getenv("MIJNHOST_TEST_DOMAIN"); v == "" {
		t.Skip("MIJNHOST_TEST_DOMAIN must be set to run acceptance tests (e.g. example.com)")
	}
}

// testAccPreCheckZone additionally requires MIJNHOST_TEST_ZONE_DOMAIN because
// the dns_zone resource performs a full record-set replace — only run it
// against a domain that is safe to overwrite completely.
func testAccPreCheckZone(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	if v := os.Getenv("MIJNHOST_TEST_ZONE_DOMAIN"); v == "" {
		t.Skip("MIJNHOST_TEST_ZONE_DOMAIN must be set to run dns_zone acceptance tests — WARNING: all records on that domain will be replaced")
	}
}

// testDomain returns the domain configured for acceptance tests.
func testDomain(t *testing.T) string {
	t.Helper()
	return os.Getenv("MIJNHOST_TEST_DOMAIN")
}

// testZoneDomain returns the domain safe for full-zone acceptance tests.
func testZoneDomain(t *testing.T) string {
	t.Helper()
	return os.Getenv("MIJNHOST_TEST_ZONE_DOMAIN")
}

// testAccCheckDestroyed verifies that the given resource no longer appears in
// state after a destroy step.
func testAccCheckDestroyed(resourceName string) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckNoResourceAttr(resourceName, "id"),
	)
}

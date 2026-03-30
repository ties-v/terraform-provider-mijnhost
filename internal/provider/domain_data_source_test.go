package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomainDataSource_basic(t *testing.T) {
	testAccPreCheck(t)
	domain := testDomain(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDataSourceConfig(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mijnhost_domain.test", "domain", domain),
					resource.TestCheckResourceAttr("data.mijnhost_domain.test", "id", domain),
					// Status must be non-empty; we don't assert a specific value
					// because it could be "active" or something else.
					resource.TestCheckResourceAttrSet("data.mijnhost_domain.test", "status"),
					resource.TestCheckResourceAttrSet("data.mijnhost_domain.test", "renewal_date"),
				),
			},
		},
	})
}

func testAccDomainDataSourceConfig(domain string) string {
	return fmt.Sprintf(`
data "mijnhost_domain" "test" {
  domain = %q
}
`, domain)
}

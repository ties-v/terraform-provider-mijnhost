package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDNSZoneResource_basic covers create, read, idempotency, update, and destroy.
//
// WARNING: This test replaces ALL DNS records on MIJNHOST_TEST_ZONE_DOMAIN.
// Only point this at a domain whose records can be safely overwritten.
func TestAccDNSZoneResource_basic(t *testing.T) {
	testAccPreCheckZone(t)
	domain := testZoneDomain(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Set a known record set and verify state.
			{
				Config: testAccDNSZoneConfig(domain, []zoneRecord{
					{Type: "A", Name: domain, Value: "192.0.2.10", TTL: 3600},
					{Type: "A", Name: "www." + domain, Value: "192.0.2.10", TTL: 3600},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mijnhost_dns_zone.test", "domain", domain),
					resource.TestCheckResourceAttr("mijnhost_dns_zone.test", "id", domain),
					resource.TestCheckResourceAttr("mijnhost_dns_zone.test", "records.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("mijnhost_dns_zone.test", "records.*", map[string]string{
						"type":  "A",
						"name":  domain,
						"value": "192.0.2.10",
						"ttl":   "3600",
					}),
				),
			},
			// Step 2: Idempotency — re-apply the same config, expect empty plan.
			{
				Config: testAccDNSZoneConfig(domain, []zoneRecord{
					{Type: "A", Name: domain, Value: "192.0.2.10", TTL: 3600},
					{Type: "A", Name: "www." + domain, Value: "192.0.2.10", TTL: 3600},
				}),
				PlanOnly: true,
			},
			// Step 3: Add a record and change a TTL — verifies UPDATE path.
			{
				Config: testAccDNSZoneConfig(domain, []zoneRecord{
					{Type: "A", Name: domain, Value: "192.0.2.10", TTL: 7200},
					{Type: "A", Name: "www." + domain, Value: "192.0.2.10", TTL: 3600},
					{Type: "TXT", Name: domain, Value: "v=spf1 ~all", TTL: 3600},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mijnhost_dns_zone.test", "records.#", "3"),
				),
			},
			// Step 4: Remove a record — verifies that the deleted record is gone.
			{
				Config: testAccDNSZoneConfig(domain, []zoneRecord{
					{Type: "A", Name: domain, Value: "192.0.2.10", TTL: 7200},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mijnhost_dns_zone.test", "records.#", "1"),
				),
			},
		},
	})
}

type zoneRecord struct {
	Type  string
	Name  string
	Value string
	TTL   int
}

func testAccDNSZoneConfig(domain string, records []zoneRecord) string {
	recordsHCL := ""
	for _, r := range records {
		recordsHCL += fmt.Sprintf(`
    {
      type  = %q
      name  = %q
      value = %q
      ttl   = %d
    },`, r.Type, r.Name, r.Value, r.TTL)
	}
	return fmt.Sprintf(`
resource "mijnhost_dns_zone" "test" {
  domain  = %q
  records = [%s
  ]
}
`, domain, recordsHCL)
}

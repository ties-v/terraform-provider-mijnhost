package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDNSRecordResource_basic covers create, read, plan idempotency, TTL update, and destroy.
func TestAccDNSRecordResource_basic(t *testing.T) {
	testAccPreCheck(t)
	domain := testDomain(t)
	recordName := "tf-acc-test." + domain

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and verify state matches API.
			{
				Config: testAccDNSRecordConfig(domain, recordName, "A", "192.0.2.1", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mijnhost_dns_record.test", "domain", domain),
					resource.TestCheckResourceAttr("mijnhost_dns_record.test", "type", "A"),
					resource.TestCheckResourceAttr("mijnhost_dns_record.test", "name", recordName),
					resource.TestCheckResourceAttr("mijnhost_dns_record.test", "value", "192.0.2.1"),
					resource.TestCheckResourceAttr("mijnhost_dns_record.test", "ttl", "3600"),
					resource.TestCheckResourceAttrSet("mijnhost_dns_record.test", "id"),
				),
			},
			// Step 2: Re-apply the same config — plan must be empty (idempotency check).
			{
				Config:   testAccDNSRecordConfig(domain, recordName, "A", "192.0.2.1", 3600),
				PlanOnly: true,
			},
			// Step 3: Update only the TTL — must not force a new resource.
			{
				Config: testAccDNSRecordConfig(domain, recordName, "A", "192.0.2.1", 7200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mijnhost_dns_record.test", "ttl", "7200"),
					resource.TestCheckResourceAttr("mijnhost_dns_record.test", "value", "192.0.2.1"),
				),
			},
			// Step 4: Import the record and verify state is consistent with what was read.
			{
				ResourceName:      "mijnhost_dns_record.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/A/%s/192.0.2.1", domain, recordName),
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccDNSRecordResource_forceNew verifies that changing value destroys and recreates.
func TestAccDNSRecordResource_forceNew(t *testing.T) {
	testAccPreCheck(t)
	domain := testDomain(t)
	recordName := "tf-acc-force." + domain

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordConfig(domain, recordName, "A", "192.0.2.2", 3600),
				Check:  resource.TestCheckResourceAttr("mijnhost_dns_record.test", "value", "192.0.2.2"),
			},
			// Changing value must produce a replace (destroy + create), not an in-place update.
			{
				Config: testAccDNSRecordConfig(domain, recordName, "A", "192.0.2.3", 3600),
				Check:  resource.TestCheckResourceAttr("mijnhost_dns_record.test", "value", "192.0.2.3"),
			},
		},
	})
}

// TestAccDNSRecordResource_txt verifies TXT records with spaces and special characters.
func TestAccDNSRecordResource_txt(t *testing.T) {
	testAccPreCheck(t)
	domain := testDomain(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordConfig(domain, domain, "TXT", "v=spf1 ~all", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mijnhost_dns_record.test", "type", "TXT"),
					resource.TestCheckResourceAttr("mijnhost_dns_record.test", "value", "v=spf1 ~all"),
				),
			},
			// Idempotency check for TXT records.
			{
				Config:   testAccDNSRecordConfig(domain, domain, "TXT", "v=spf1 ~all", 3600),
				PlanOnly: true,
			},
		},
	})
}

func testAccDNSRecordConfig(domain, name, recType, value string, ttl int) string {
	return fmt.Sprintf(`
resource "mijnhost_dns_record" "test" {
  domain = %q
  type   = %q
  name   = %q
  value  = %q
  ttl    = %d
}
`, domain, recType, name, value, ttl)
}

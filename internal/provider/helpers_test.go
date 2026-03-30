package provider

// Unit tests for unexported helpers. These run in the provider package so they
// can access package-private functions directly, without needing a real API.

import (
	"context"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ties-v/terraform-provider-mijnhost/internal/mijnhost"
)

// --- findRecord ---

func TestFindRecord_found(t *testing.T) {
	records := []mijnhost.DNSRecord{
		{Type: "A", Name: "example.com", Value: "1.2.3.4", TTL: 3600},
		{Type: "MX", Name: "example.com", Value: "10 mail.example.com", TTL: 3600},
	}

	got := findRecord(records, "MX", "example.com", "10 mail.example.com")
	if got == nil {
		t.Fatal("expected record to be found, got nil")
	}
	if got.TTL != 3600 {
		t.Errorf("TTL = %d, want 3600", got.TTL)
	}
}

func TestFindRecord_notFound(t *testing.T) {
	records := []mijnhost.DNSRecord{
		{Type: "A", Name: "example.com", Value: "1.2.3.4", TTL: 3600},
	}

	if got := findRecord(records, "AAAA", "example.com", "::1"); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestFindRecord_emptySlice(t *testing.T) {
	if got := findRecord(nil, "A", "example.com", "1.2.3.4"); got != nil {
		t.Errorf("expected nil for empty slice, got %+v", got)
	}
}

// --- recordID ---

func TestRecordID(t *testing.T) {
	id := recordID("example.com", "A", "www.example.com", "1.2.3.4")
	want := "example.com/A/www.example.com/1.2.3.4"
	if id != want {
		t.Errorf("recordID = %q, want %q", id, want)
	}
}

// --- recordsToState / recordsFromState round-trip ---

func TestRecordsStateRoundTrip(t *testing.T) {
	ctx := context.Background()

	input := []mijnhost.DNSRecord{
		{Type: "A", Name: "example.com", Value: "1.2.3.4", TTL: 3600},
		{Type: "TXT", Name: "example.com", Value: "v=spf1 ~all", TTL: 600},
	}

	setVal, diags := recordsToState(ctx, input)
	if diags.HasError() {
		t.Fatalf("recordsToState error: %v", diags)
	}

	output, diags := recordsFromState(ctx, setVal)
	if diags.HasError() {
		t.Fatalf("recordsFromState error: %v", diags)
	}

	if len(output) != len(input) {
		t.Fatalf("len = %d, want %d", len(output), len(input))
	}

	// Sets are unordered, so sort both slices before comparing.
	sortRecords := func(r []mijnhost.DNSRecord) {
		sort.Slice(r, func(i, j int) bool {
			return r[i].Type+r[i].Name+r[i].Value < r[j].Type+r[j].Name+r[j].Value
		})
	}
	sortRecords(input)
	sortRecords(output)

	for i := range input {
		if output[i] != input[i] {
			t.Errorf("record[%d]: got %+v, want %+v", i, output[i], input[i])
		}
	}
}

func TestRecordsToState_empty(t *testing.T) {
	ctx := context.Background()

	setVal, diags := recordsToState(ctx, []mijnhost.DNSRecord{})
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}

	var models []dnsRecordModel
	diags = setVal.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs error: %v", diags)
	}
	if len(models) != 0 {
		t.Errorf("expected 0 elements, got %d", len(models))
	}
}

// --- dnsRecordAttrTypes ---

func TestDNSRecordAttrTypes(t *testing.T) {
	_, diags := types.ObjectValue(dnsRecordAttrTypes, map[string]attr.Value{
		"type":  types.StringValue("A"),
		"name":  types.StringValue("example.com"),
		"value": types.StringValue("1.2.3.4"),
		"ttl":   types.Int64Value(3600),
	})
	if diags.HasError() {
		t.Errorf("ObjectValue with dnsRecordAttrTypes failed: %v", diags)
	}
}

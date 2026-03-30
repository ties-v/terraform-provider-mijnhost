package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ties-v/terraform-provider-mijnhost/internal/mijnhost"
)

var _ resource.Resource = &DNSRecordResource{}
var _ resource.ResourceWithImportState = &DNSRecordResource{}

type DNSRecordResource struct {
	client *mijnhost.Client
}

type DNSRecordResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
	Type   types.String `tfsdk:"type"`
	Name   types.String `tfsdk:"name"`
	Value  types.String `tfsdk:"value"`
	TTL    types.Int64  `tfsdk:"ttl"`
}

func NewDNSRecordResource() resource.Resource {
	return &DNSRecordResource{}
}

func (r *DNSRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *DNSRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a single DNS record for a mijn.host domain. Changes to domain, type, name, or value will force a new resource. Only TTL can be updated in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this DNS record, in the format domain/type/name/value.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "The domain name (e.g. example.com).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "DNS record type (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA, etc.).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "DNS record name/hostname. Use the bare domain name for apex records (e.g. example.com), or a subdomain (e.g. www.example.com).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:    true,
				Description: "DNS record value (e.g. an IP address for A records, or the target for CNAME records).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.Int64Attribute{
				Required:    true,
				Description: "Time to live in seconds.",
			},
		},
	}
}

func (r *DNSRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*mijnhost.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *mijnhost.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *DNSRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DNSRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()

	// Serialize all read-modify-write operations on the same domain.
	unlock := r.client.LockDomain(domain)
	defer unlock()

	// Read the current full record set.
	records, err := r.client.GetDNSRecords(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading DNS Records", fmt.Sprintf("Could not read DNS records for domain %q: %s", domain, err))
		return
	}

	// Check if a record with this type+name+value already exists.
	newRecord := mijnhost.DNSRecord{
		Type:  data.Type.ValueString(),
		Name:  data.Name.ValueString(),
		Value: data.Value.ValueString(),
		TTL:   data.TTL.ValueInt64(),
	}

	for _, r := range records {
		if r.Type == newRecord.Type && r.Name == newRecord.Name && r.Value == newRecord.Value {
			resp.Diagnostics.AddError(
				"DNS Record Already Exists",
				fmt.Sprintf("A DNS record with type=%q name=%q value=%q already exists for domain %q.", newRecord.Type, newRecord.Name, newRecord.Value, domain),
			)
			return
		}
	}

	// Append the new record and write the full set back.
	records = append(records, newRecord)
	if err := r.client.UpdateDNSRecords(ctx, domain, records); err != nil {
		resp.Diagnostics.AddError("Error Creating DNS Record", fmt.Sprintf("Could not create DNS record: %s", err))
		return
	}

	data.ID = types.StringValue(recordID(domain, newRecord.Type, newRecord.Name, newRecord.Value))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DNSRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()
	records, err := r.client.GetDNSRecords(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading DNS Records", fmt.Sprintf("Could not read DNS records for domain %q: %s", domain, err))
		return
	}

	found := findRecord(records, data.Type.ValueString(), data.Name.ValueString(), data.Value.ValueString())
	if found == nil {
		// Record no longer exists remotely.
		resp.State.RemoveResource(ctx)
		return
	}

	data.TTL = types.Int64Value(found.TTL)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DNSRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only TTL can be updated (type/name/value are ForceNew).
	// Use PATCH to update the single record.
	record := mijnhost.DNSRecord{
		Type:  data.Type.ValueString(),
		Name:  data.Name.ValueString(),
		Value: data.Value.ValueString(),
		TTL:   data.TTL.ValueInt64(),
	}

	if err := r.client.PatchDNSRecord(ctx, data.Domain.ValueString(), record); err != nil {
		resp.Diagnostics.AddError("Error Updating DNS Record", fmt.Sprintf("Could not update DNS record: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DNSRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()

	// Serialize all read-modify-write operations on the same domain.
	unlock := r.client.LockDomain(domain)
	defer unlock()

	records, err := r.client.GetDNSRecords(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading DNS Records", fmt.Sprintf("Could not read DNS records for domain %q: %s", domain, err))
		return
	}

	// Filter out the record to delete.
	filtered := make([]mijnhost.DNSRecord, 0, len(records))
	for _, rec := range records {
		if rec.Type == data.Type.ValueString() && rec.Name == data.Name.ValueString() && rec.Value == data.Value.ValueString() {
			continue
		}
		filtered = append(filtered, rec)
	}

	if len(filtered) == len(records) {
		// Record not found — already gone, treat as success.
		return
	}

	if err := r.client.UpdateDNSRecords(ctx, domain, filtered); err != nil {
		resp.Diagnostics.AddError("Error Deleting DNS Record", fmt.Sprintf("Could not delete DNS record: %s", err))
		return
	}
}

func recordID(domain, recType, name, value string) string {
	return fmt.Sprintf("%s/%s/%s/%s", domain, recType, name, value)
}

func findRecord(records []mijnhost.DNSRecord, recType, name, value string) *mijnhost.DNSRecord {
	for i := range records {
		if records[i].Type == recType && records[i].Name == name && records[i].Value == value {
			return &records[i]
		}
	}
	return nil
}

// ImportState supports `tofu import mijnhost_dns_record.x domain/TYPE/name/value`.
// The value field is last and may itself contain slashes.
func (r *DNSRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format: domain/TYPE/name/value — got %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), parts[3])...)
}

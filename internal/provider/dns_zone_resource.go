package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ties-v/terraform-provider-mijnhost/internal/mijnhost"
)

var _ resource.Resource = &DNSZoneResource{}

// DNSZoneResource manages the complete set of DNS records for a domain.
type DNSZoneResource struct {
	client *mijnhost.Client
}

// DNSZoneResourceModel holds the Terraform state for a managed DNS zone.
type DNSZoneResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Domain  types.String `tfsdk:"domain"`
	Records types.List   `tfsdk:"records"`
}

// dnsRecordModel is the nested record object within the zone.
type dnsRecordModel struct {
	Type  types.String `tfsdk:"type"`
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	TTL   types.Int64  `tfsdk:"ttl"`
}

var dnsRecordAttrTypes = map[string]attr.Type{
	"type":  types.StringType,
	"name":  types.StringType,
	"value": types.StringType,
	"ttl":   types.Int64Type,
}

func NewDNSZoneResource() resource.Resource {
	return &DNSZoneResource{}
}

func (r *DNSZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

func (r *DNSZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the complete set of DNS records for a mijn.host domain. " +
			"Any records not listed in this resource will be removed. " +
			"Use mijnhost_dns_record for managing individual records without affecting others.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The domain name (used as the resource ID).",
			},
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "The domain name to manage DNS records for (e.g. example.com).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"records": schema.ListNestedAttribute{
				Required:    true,
				Description: "The complete set of DNS records for the domain. All existing records not listed here will be deleted.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "DNS record type (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA, etc.).",
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "DNS record name/hostname.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "DNS record value.",
						},
						"ttl": schema.Int64Attribute{
							Required:    true,
							Description: "Time to live in seconds.",
						},
					},
				},
			},
		},
	}
}

func (r *DNSZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DNSZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DNSZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, diags := recordsFromState(ctx, data.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()
	if err := r.client.UpdateDNSRecords(ctx, domain, records); err != nil {
		resp.Diagnostics.AddError("Error Creating DNS Zone", fmt.Sprintf("Could not set DNS records for domain %q: %s", domain, err))
		return
	}

	data.ID = types.StringValue(domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DNSZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()
	records, err := r.client.GetDNSRecords(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading DNS Zone", fmt.Sprintf("Could not read DNS records for domain %q: %s", domain, err))
		return
	}

	listVal, diags := recordsToState(ctx, records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Records = listVal
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DNSZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, diags := recordsFromState(ctx, data.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()
	if err := r.client.UpdateDNSRecords(ctx, domain, records); err != nil {
		resp.Diagnostics.AddError("Error Updating DNS Zone", fmt.Sprintf("Could not update DNS records for domain %q: %s", domain, err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSZoneResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// The mijn.host API does not support an empty record set, so deleting this resource
	// only removes it from Terraform state. The DNS records remain in mijn.host.
}

// recordsFromState converts a types.List of records into []mijnhost.DNSRecord.
func recordsFromState(ctx context.Context, list types.List) ([]mijnhost.DNSRecord, diag.Diagnostics) {
	var models []dnsRecordModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}

	records := make([]mijnhost.DNSRecord, len(models))
	for i, m := range models {
		records[i] = mijnhost.DNSRecord{
			Type:  m.Type.ValueString(),
			Name:  m.Name.ValueString(),
			Value: m.Value.ValueString(),
			TTL:   m.TTL.ValueInt64(),
		}
	}
	return records, diags
}

// recordsToState converts []mijnhost.DNSRecord into a types.List suitable for Terraform state.
func recordsToState(ctx context.Context, records []mijnhost.DNSRecord) (types.List, diag.Diagnostics) {
	objectType := types.ObjectType{AttrTypes: dnsRecordAttrTypes}

	elements := make([]attr.Value, len(records))
	for i, rec := range records {
		obj, diags := types.ObjectValue(dnsRecordAttrTypes, map[string]attr.Value{
			"type":  types.StringValue(rec.Type),
			"name":  types.StringValue(rec.Name),
			"value": types.StringValue(rec.Value),
			"ttl":   types.Int64Value(rec.TTL),
		})
		if diags.HasError() {
			return types.ListNull(objectType), diags
		}
		elements[i] = obj
	}

	return types.ListValue(objectType, elements)
}

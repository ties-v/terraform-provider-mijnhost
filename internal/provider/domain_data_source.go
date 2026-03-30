package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ties-v/terraform-provider-mijnhost/internal/mijnhost"
)

var _ datasource.DataSource = &DomainDataSource{}

// DomainDataSource provides read-only access to mijn.host domain info.
type DomainDataSource struct {
	client *mijnhost.Client
}

// DomainDataSourceModel holds the Terraform state for the domain data source.
type DomainDataSourceModel struct {
	Domain      types.String `tfsdk:"domain"`
	ID          types.String `tfsdk:"id"`
	RenewalDate types.String `tfsdk:"renewal_date"`
	Status      types.String `tfsdk:"status"`
	Tags        types.List   `tfsdk:"tags"`
}

func NewDomainDataSource() datasource.DataSource {
	return &DomainDataSource{}
}

func (d *DomainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *DomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a mijn.host domain.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "The domain name to look up (e.g. example.com).",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The domain name (used as the data source ID).",
			},
			"renewal_date": schema.StringAttribute{
				Computed:    true,
				Description: "The date on which the domain will be renewed (YYYY-MM-DD).",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The current status of the domain (e.g. active, Cancelled).",
			},
			"tags": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the domain.",
			},
		},
	}
}

func (d *DomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*mijnhost.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *mijnhost.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DomainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := d.client.GetDomain(ctx, data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Domain", fmt.Sprintf("Could not read domain %q: %s", data.Domain.ValueString(), err))
		return
	}

	data.ID = types.StringValue(domain.Domain)
	data.RenewalDate = types.StringValue(domain.RenewalDate)
	data.Status = types.StringValue(domain.Status)

	tags := make([]string, len(domain.Tags))
	copy(tags, domain.Tags)
	tagList, diags := types.ListValueFrom(ctx, types.StringType, tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Tags = tagList
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

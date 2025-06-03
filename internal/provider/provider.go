// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &goodAccessProvider{
			version: version,
		}
	}
}

type goodAccessProvider struct {
	version string
}

type goodAccessProviderModel struct {
	Token types.String `tfsdk:"token"`
}

func (p *goodAccessProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config goodAccessProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	resp.ResourceData = config
}

func (p *goodAccessProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "goodaccess"
	resp.Version = p.version
}

func (p *goodAccessProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Required:    true,
				Description: "GoodAccess API token",
			},
		},
	}
}

func (p *goodAccessProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSystemResource,
		NewAccessCardResource,
		NewRelationACSResource,
	}
}

func (p *goodAccessProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

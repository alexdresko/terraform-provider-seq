package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = (*SeqProvider)(nil)

// SeqProvider implements the Terraform provider for Seq.
//
// Configuration is passed via provider schema or env vars (see schema descriptions).
// The provider supplies a configured *Client to resources and datasources.
type SeqProvider struct {
	version string
}

// SeqProviderModel describes the provider data model.
//
// Provider configuration can also be set using env vars:
// - SEQ_SERVER_URL
// - SEQ_API_KEY
// - SEQ_INSECURE_SKIP_VERIFY
// - SEQ_TIMEOUT_SECONDS
type SeqProviderModel struct {
	ServerURL          types.String `tfsdk:"server_url"`
	APIKey             types.String `tfsdk:"api_key"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	TimeoutSeconds     types.Int64  `tfsdk:"timeout_seconds"`
}

// New creates a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SeqProvider{version: version}
	}
}

func (p *SeqProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "seq"
	resp.Version = p.version
}

func (p *SeqProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Seq resources via the Seq HTTP API.",
		Attributes: map[string]schema.Attribute{
			"server_url": schema.StringAttribute{
				Description: "Base URL for the Seq server, e.g. https://seq.example.com or http://localhost:5342. Can be set via SEQ_SERVER_URL.",
				Optional:    true,
				Validators: []frameworkvalidator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"api_key": schema.StringAttribute{
				Description: "Seq API key used for authentication. Sent as the X-Seq-ApiKey header. Can be set via SEQ_API_KEY.",
				Optional:    true,
				Sensitive:   true,
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Description: "Skip TLS certificate verification (NOT recommended). Can be set via SEQ_INSECURE_SKIP_VERIFY.",
				Optional:    true,
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "HTTP client timeout in seconds. Can be set via SEQ_TIMEOUT_SECONDS.",
				Optional:    true,
			},
		},
	}
}

func (p *SeqProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config SeqProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := NewClientFromConfig(ctx, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Configured Seq provider client")

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *SeqProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAPIKeyResource,
	}
}

func (p *SeqProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHealthDataSource,
	}
}

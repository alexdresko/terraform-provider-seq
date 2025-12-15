package provider

import (
	"context"
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworkvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*APIKeyResource)(nil)
var _ resource.ResourceWithConfigure = (*APIKeyResource)(nil)
var _ resource.ResourceWithImportState = (*APIKeyResource)(nil)

// APIKeyResource manages Seq API keys via /api/apikeys.
//
// Ref: https://datalust.co/docs/server-http-api#api-apikeys
type APIKeyResource struct {
	client *Client
}

// APIKeyModel is the Terraform state model for an API key.
type APIKeyModel struct {
	ID          types.String `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	Token       types.String `tfsdk:"token"`
	OwnerID     types.String `tfsdk:"owner_id"`
	Permissions types.Set    `tfsdk:"permissions"`
}

func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

func (r *APIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Seq API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Seq API key id.",
				Computed:    true,
			},
			"title": schema.StringAttribute{
				Description: "Human-friendly title for the API key.",
				Required:    true,
				Validators: []frameworkvalidator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"token": schema.StringAttribute{
				Description: "The API key token/secret. Seq may only return this on create; it is stored in state as sensitive.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					// If Seq does not return the token on reads, keep the existing value.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owner_id": schema.StringAttribute{
				Description: "Owner principal id. Depending on permissions, you may only be able to set this to yourself.",
				Optional:    true,
				Computed:    true,
			},
			"permissions": schema.SetAttribute{
				Description: "Permissions delegated to the API key (e.g. Read, Write, Ingest, Project, System).",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *APIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *provider.Client, got a different type.",
		)
		return
	}
	r.client = client
}

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.checkConfigured(&resp.Diagnostics) {
		return
	}

	var plan APIKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := apiKeyRequestBody(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created apiKeyResponse
	if err := r.client.doJSON(ctx, http.MethodPost, "/api/apikeys", body, &created); err != nil {
		resp.Diagnostics.AddError("Failed to create Seq API key", err.Error())
		return
	}

	state := plan
	applyAPIKeyResponse(&state, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.checkConfigured(&resp.Diagnostics) {
		return
	}

	var state APIKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		resp.State.RemoveResource(ctx)
		return
	}

	var got apiKeyResponse
	path := "/api/apikeys/" + state.ID.ValueString()
	if err := r.client.doJSON(ctx, http.MethodGet, path, nil, &got); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Seq API key", err.Error())
		return
	}

	newState := state
	applyAPIKeyResponse(&newState, got)

	// Seq may omit token on read; keep previous.
	if got.Token == "" {
		newState.Token = state.Token
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.checkConfigured(&resp.Diagnostics) {
		return
	}

	var plan APIKeyModel
	var state APIKeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		resp.Diagnostics.AddError("Missing id", "Cannot update API key without an id in state")
		return
	}

	body, diags := apiKeyRequestBody(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var updated apiKeyResponse
	path := "/api/apikeys/" + state.ID.ValueString()
	if err := r.client.doJSON(ctx, http.MethodPut, path, body, &updated); err != nil {
		resp.Diagnostics.AddError("Failed to update Seq API key", err.Error())
		return
	}

	newState := plan
	newState.ID = state.ID
	applyAPIKeyResponse(&newState, updated)

	// Token may not be returned on update; keep previous.
	if updated.Token == "" {
		newState.Token = state.Token
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.checkConfigured(&resp.Diagnostics) {
		return
	}

	var state APIKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		return
	}

	path := "/api/apikeys/" + state.ID.ValueString()
	if err := r.client.doJSON(ctx, http.MethodDelete, path, nil, nil); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Failed to delete Seq API key", err.Error())
		return
	}
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiKeyResponse struct {
	ID          string   `json:"Id"`
	Title       string   `json:"Title"`
	Token       string   `json:"Token"`
	OwnerID     string   `json:"OwnerId"`
	Permissions []string `json:"Permissions"`
}

func apiKeyRequestBody(ctx context.Context, plan APIKeyModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := map[string]any{
		"Title": plan.Title.ValueString(),
	}

	if !plan.OwnerID.IsNull() && !plan.OwnerID.IsUnknown() && plan.OwnerID.ValueString() != "" {
		body["OwnerId"] = plan.OwnerID.ValueString()
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		diags.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body["Permissions"] = perms
	}

	return body, diags
}

func applyAPIKeyResponse(state *APIKeyModel, resp apiKeyResponse) {
	if resp.ID != "" {
		state.ID = types.StringValue(resp.ID)
	}
	if resp.Title != "" {
		state.Title = types.StringValue(resp.Title)
	}
	if resp.Token != "" {
		state.Token = types.StringValue(resp.Token)
	}
	if resp.OwnerID != "" {
		state.OwnerID = types.StringValue(resp.OwnerID)
	}
	if resp.Permissions != nil {
		state.Permissions = types.SetValueMust(types.StringType, stringSliceToAttrValues(resp.Permissions))
	}
}

func stringSliceToAttrValues(vs []string) []attr.Value {
	out := make([]attr.Value, 0, len(vs))
	for _, v := range vs {
		out = append(out, types.StringValue(v))
	}
	return out
}

var errNotConfigured = errors.New("provider not configured")

func (r *APIKeyResource) checkConfigured(respDiags *diag.Diagnostics) bool {
	if r.client == nil {
		respDiags.AddError("Provider not configured", errNotConfigured.Error())
		return false
	}
	return true
}

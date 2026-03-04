package provider

import (
	"context"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/domain"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithConfigure   = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

type DomainResource struct {
	configured bool
}

type DomainResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	IsSatellite       types.Bool   `tfsdk:"is_satellite"`
	ProxyURL          types.String `tfsdk:"proxy_url"`
	IsSecondary       types.Bool   `tfsdk:"is_secondary"`
	FrontendAPIURL    types.String `tfsdk:"frontend_api_url"`
	AccountPortalURL  types.String `tfsdk:"account_portal_url"`
	DevelopmentOrigin types.String `tfsdk:"development_origin"`
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	_, ok := req.ProviderData.(string)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected string (API key), got something else. Please report this issue.",
		)
		return
	}
	r.configured = true
}

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk domain. Use this to add satellite domains for multi-domain authentication or to configure production domains for your applications.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The domain name (e.g. \"app.example.com\"). Can include port for development instances (e.g. \"localhost:3000\").",
			},
			"is_satellite": schema.BoolAttribute{
				Required:    true,
				Description: "Whether this is a satellite domain. Instances can have only one primary domain, so this must be true when adding additional domains.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"proxy_url": schema.StringAttribute{
				Optional:    true,
				Description: "The full URL of the proxy to use for this domain. Can only be updated for production instances.",
			},
			"is_secondary": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this is a secondary domain for multi-app support on the same root domain.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"frontend_api_url": schema.StringAttribute{
				Computed:    true,
				Description: "The Frontend API URL for this domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_portal_url": schema.StringAttribute{
				Computed:    true,
				Description: "The Accounts Portal URL for this domain.",
			},
			"development_origin": schema.StringAttribute{
				Computed:    true,
				Description: "The development origin for this domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &domain.CreateParams{
		Name:        clerkgo.String(plan.Name.ValueString()),
		IsSatellite: clerkgo.Bool(plan.IsSatellite.ValueBool()),
	}

	if !plan.ProxyURL.IsNull() && !plan.ProxyURL.IsUnknown() {
		params.ProxyURL = clerkgo.String(plan.ProxyURL.ValueString())
	}

	d, err := domain.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create domain", err.Error())
		return
	}

	mapDomainResponseToModel(d, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created domain", map[string]any{"id": d.ID, "name": d.Name})
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No Get endpoint — must list all and find by ID
	list, err := domain.List(ctx, &domain.ListParams{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to list domains", err.Error())
		return
	}

	var found *clerkgo.Domain
	for _, d := range list.Domains {
		if d.ID == state.ID.ValueString() {
			found = d
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapDomainResponseToModel(found, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DomainResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &domain.UpdateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
	}

	if !plan.ProxyURL.IsNull() && !plan.ProxyURL.IsUnknown() {
		params.ProxyURL = clerkgo.String(plan.ProxyURL.ValueString())
	} else {
		params.ProxyURL = clerkgo.String("")
	}

	if !plan.IsSecondary.IsNull() && !plan.IsSecondary.IsUnknown() {
		params.IsSecondary = clerkgo.Bool(plan.IsSecondary.ValueBool())
	}

	d, err := domain.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update domain", err.Error())
		return
	}

	mapDomainResponseToModel(d, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated domain", map[string]any{"id": d.ID, "name": d.Name})
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := domain.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete domain",
			fmt.Sprintf("Could not delete domain ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted domain", map[string]any{"id": state.ID.ValueString()})
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapDomainResponseToModel(d *clerkgo.Domain, model *DomainResourceModel) {
	model.ID = types.StringValue(d.ID)
	model.Name = types.StringValue(d.Name)
	model.IsSatellite = types.BoolValue(d.IsSatellite)
	model.FrontendAPIURL = types.StringValue(d.FrontendAPIURL)
	model.DevelopmentOrigin = types.StringValue(d.DevelopmentOrigin)

	if d.AccountPortalURL != nil && *d.AccountPortalURL != "" {
		model.AccountPortalURL = types.StringValue(*d.AccountPortalURL)
	} else {
		model.AccountPortalURL = types.StringNull()
	}

	if d.ProxyURL != nil && *d.ProxyURL != "" {
		model.ProxyURL = types.StringValue(*d.ProxyURL)
	}

	// is_secondary is not returned by the Domain API type — default to false
	// so the computed value is always known after apply
	if model.IsSecondary.IsUnknown() {
		model.IsSecondary = types.BoolValue(false)
	}
}

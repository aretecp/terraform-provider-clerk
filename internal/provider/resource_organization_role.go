package provider

import (
	"context"
	"fmt"
	"time"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/organizationrole"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &OrganizationRoleResource{}
	_ resource.ResourceWithConfigure   = &OrganizationRoleResource{}
	_ resource.ResourceWithImportState = &OrganizationRoleResource{}
)

func NewOrganizationRoleResource() resource.Resource {
	return &OrganizationRoleResource{}
}

type OrganizationRoleResource struct {
	configured bool
}

type OrganizationRoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Key         types.String `tfsdk:"key"`
	Description types.String `tfsdk:"description"`
	Permissions types.Set    `tfsdk:"permissions"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *OrganizationRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role"
}

func (r *OrganizationRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk organization role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the role (e.g. \"Internal Admin\").",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "Unique key for the role (e.g. \"internal_admin\").",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the role.",
			},
			"permissions": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Set of permission IDs assigned to this role.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was last updated.",
			},
		},
	}
}

func (r *OrganizationRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationrole.CreateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
		Key:  clerkgo.String(plan.Key.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var permKeys []string
		diags = plan.Permissions.ElementsAs(ctx, &permKeys, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.Permissions = &permKeys
	}

	role, err := organizationrole.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create organization role", err.Error())
		return
	}

	mapOrgRoleResponseToModel(role, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created organization role", map[string]any{"id": role.ID})
}

func (r *OrganizationRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := organizationrole.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read organization role",
			fmt.Sprintf("Could not read role ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapOrgRoleResponseToModel(role, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OrganizationRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationRoleResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationrole.UpdateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
		Key:  clerkgo.String(plan.Key.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var permKeys []string
		diags = plan.Permissions.ElementsAs(ctx, &permKeys, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.Permissions = &permKeys
	} else {
		// Explicitly set empty permissions to clear them
		empty := []string{}
		params.Permissions = &empty
	}

	role, err := organizationrole.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update organization role", err.Error())
		return
	}

	mapOrgRoleResponseToModel(role, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated organization role", map[string]any{"id": role.ID})
}

func (r *OrganizationRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := organizationrole.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete organization role",
			fmt.Sprintf("Could not delete role ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted organization role", map[string]any{"id": state.ID.ValueString()})
}

func (r *OrganizationRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapOrgRoleResponseToModel(role *clerkgo.OrganizationRole, model *OrganizationRoleResourceModel) {
	model.ID = types.StringValue(role.ID)
	model.Name = types.StringValue(role.Name)
	model.Key = types.StringValue(role.Key)
	model.CreatedAt = types.StringValue(time.UnixMilli(role.CreatedAt).UTC().Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(time.UnixMilli(role.UpdatedAt).UTC().Format(time.RFC3339))

	if role.Description != nil {
		model.Description = types.StringValue(*role.Description)
	}

	// Extract permission IDs from the embedded permission objects
	if len(role.Permissions) > 0 {
		permIDs := make([]string, len(role.Permissions))
		for i, p := range role.Permissions {
			permIDs[i] = p.ID
		}
		permSet, _ := types.SetValueFrom(context.Background(), types.StringType, permIDs)
		model.Permissions = permSet
	} else {
		model.Permissions = types.SetNull(types.StringType)
	}
}

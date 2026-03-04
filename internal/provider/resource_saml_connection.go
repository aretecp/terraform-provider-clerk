package provider

import (
	"context"
	"fmt"
	"time"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/samlconnection"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &SAMLConnectionResource{}
	_ resource.ResourceWithConfigure   = &SAMLConnectionResource{}
	_ resource.ResourceWithImportState = &SAMLConnectionResource{}
)

func NewSAMLConnectionResource() resource.Resource {
	return &SAMLConnectionResource{}
}

type SAMLConnectionResource struct {
	configured bool
}

type SAMLConnectionAttributeMappingModel struct {
	UserID       types.String `tfsdk:"user_id"`
	EmailAddress types.String `tfsdk:"email_address"`
	FirstName    types.String `tfsdk:"first_name"`
	LastName     types.String `tfsdk:"last_name"`
}

type SAMLConnectionResourceModel struct {
	ID                               types.String                         `tfsdk:"id"`
	Name                             types.String                         `tfsdk:"name"`
	Domain                           types.String                         `tfsdk:"domain"`
	SamlProvider                     types.String                         `tfsdk:"saml_provider"`
	OrganizationID                   types.String                         `tfsdk:"organization_id"`
	IdpEntityID                      types.String                         `tfsdk:"idp_entity_id"`
	IdpSsoURL                        types.String                         `tfsdk:"idp_sso_url"`
	IdpCertificate                   types.String                         `tfsdk:"idp_certificate"`
	IdpMetadataURL                   types.String                         `tfsdk:"idp_metadata_url"`
	IdpMetadata                      types.String                         `tfsdk:"idp_metadata"`
	Active                           types.Bool                           `tfsdk:"active"`
	SyncUserAttributes               types.Bool                           `tfsdk:"sync_user_attributes"`
	AllowSubdomains                  types.Bool                           `tfsdk:"allow_subdomains"`
	AllowIdpInitiated                types.Bool                           `tfsdk:"allow_idp_initiated"`
	DisableAdditionalIdentifications types.Bool                           `tfsdk:"disable_additional_identifications"`
	AttributeMapping                 *SAMLConnectionAttributeMappingModel `tfsdk:"attribute_mapping"`
	AcsURL                           types.String                         `tfsdk:"acs_url"`
	SPEntityID                       types.String                         `tfsdk:"sp_entity_id"`
	SPMetadataURL                    types.String                         `tfsdk:"sp_metadata_url"`
	UserCount                        types.Int64                          `tfsdk:"user_count"`
	CreatedAt                        types.String                         `tfsdk:"created_at"`
	UpdatedAt                        types.String                         `tfsdk:"updated_at"`
}

func (r *SAMLConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SAMLConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_connection"
}

func (r *SAMLConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk SAML connection for SSO authentication (e.g. Microsoft Entra ID).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the SAML connection.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the SAML connection (e.g. \"Microsoft Entra ID\").",
			},
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "Domain for the SAML connection (e.g. \"aretecp.com\").",
			},
			"saml_provider": schema.StringAttribute{
				Required:    true,
				Description: "SAML provider type (e.g. \"saml_microsoft\", \"saml_custom\"). Cannot be changed after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Description: "Clerk organization ID to associate with this SAML connection.",
			},
			"idp_entity_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Identity Provider Entity ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"idp_sso_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Identity Provider Single Sign-On URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"idp_certificate": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Identity Provider X.509 certificate.",
			},
			"idp_metadata_url": schema.StringAttribute{
				Optional:    true,
				Description: "Identity Provider federation metadata URL for auto-configuration.",
			},
			"idp_metadata": schema.StringAttribute{
				Optional:    true,
				Description: "Raw Identity Provider SAML metadata XML.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the SAML connection is active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"sync_user_attributes": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to sync user attributes from the IdP on each sign-in.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_subdomains": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to allow subdomains of the configured domain.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_idp_initiated": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to allow IdP-initiated SSO flows.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"disable_additional_identifications": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to disable additional identification methods for SAML users.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"attribute_mapping": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "SAML attribute mapping configuration.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"user_id": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "SAML attribute for user ID.",
					},
					"email_address": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "SAML attribute for email address.",
					},
					"first_name": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "SAML attribute for first name.",
					},
					"last_name": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "SAML attribute for last name.",
					},
				},
			},
			"acs_url": schema.StringAttribute{
				Computed:    true,
				Description: "Assertion Consumer Service URL. Configure this in your Identity Provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sp_entity_id": schema.StringAttribute{
				Computed:    true,
				Description: "Service Provider Entity ID. Configure this in your Identity Provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sp_metadata_url": schema.StringAttribute{
				Computed:    true,
				Description: "Service Provider metadata URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of users associated with this SAML connection.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the SAML connection was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the SAML connection was last updated.",
			},
		},
	}
}

func (r *SAMLConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SAMLConnectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &samlconnection.CreateParams{
		Name:     clerkgo.String(plan.Name.ValueString()),
		Domain:   clerkgo.String(plan.Domain.ValueString()),
		Provider: clerkgo.String(plan.SamlProvider.ValueString()),
	}

	if !plan.OrganizationID.IsNull() && !plan.OrganizationID.IsUnknown() {
		params.OrganizationID = clerkgo.String(plan.OrganizationID.ValueString())
	}
	if !plan.IdpEntityID.IsNull() && !plan.IdpEntityID.IsUnknown() {
		params.IdpEntityID = clerkgo.String(plan.IdpEntityID.ValueString())
	}
	if !plan.IdpSsoURL.IsNull() && !plan.IdpSsoURL.IsUnknown() {
		params.IdpSsoURL = clerkgo.String(plan.IdpSsoURL.ValueString())
	}
	if !plan.IdpCertificate.IsNull() && !plan.IdpCertificate.IsUnknown() {
		params.IdpCertificate = clerkgo.String(plan.IdpCertificate.ValueString())
	}
	if !plan.IdpMetadataURL.IsNull() && !plan.IdpMetadataURL.IsUnknown() {
		params.IdpMetadataURL = clerkgo.String(plan.IdpMetadataURL.ValueString())
	}
	if !plan.IdpMetadata.IsNull() && !plan.IdpMetadata.IsUnknown() {
		params.IdpMetadata = clerkgo.String(plan.IdpMetadata.ValueString())
	}

	if plan.AttributeMapping != nil {
		mapping := &samlconnection.AttributeMappingParams{}
		if !plan.AttributeMapping.UserID.IsNull() && !plan.AttributeMapping.UserID.IsUnknown() {
			mapping.UserID = plan.AttributeMapping.UserID.ValueString()
		}
		if !plan.AttributeMapping.EmailAddress.IsNull() && !plan.AttributeMapping.EmailAddress.IsUnknown() {
			mapping.EmailAddress = plan.AttributeMapping.EmailAddress.ValueString()
		}
		if !plan.AttributeMapping.FirstName.IsNull() && !plan.AttributeMapping.FirstName.IsUnknown() {
			mapping.FirstName = plan.AttributeMapping.FirstName.ValueString()
		}
		if !plan.AttributeMapping.LastName.IsNull() && !plan.AttributeMapping.LastName.IsUnknown() {
			mapping.LastName = plan.AttributeMapping.LastName.ValueString()
		}
		params.AttributeMapping = mapping
	}

	conn, err := samlconnection.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create SAML connection", err.Error())
		return
	}

	mapSAMLConnectionResponseToModel(conn, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created SAML connection", map[string]any{"id": conn.ID})
}

func (r *SAMLConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SAMLConnectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := samlconnection.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read SAML connection",
			fmt.Sprintf("Could not read SAML connection ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapSAMLConnectionResponseToModel(conn, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *SAMLConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SAMLConnectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SAMLConnectionResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &samlconnection.UpdateParams{
		Name:   clerkgo.String(plan.Name.ValueString()),
		Domain: clerkgo.String(plan.Domain.ValueString()),
	}

	if !plan.OrganizationID.IsNull() && !plan.OrganizationID.IsUnknown() {
		params.OrganizationID = clerkgo.String(plan.OrganizationID.ValueString())
	} else {
		// Explicitly unset organization_id if removed from config
		params.OrganizationID = clerkgo.String("")
	}

	if !plan.IdpEntityID.IsNull() && !plan.IdpEntityID.IsUnknown() {
		params.IdpEntityID = clerkgo.String(plan.IdpEntityID.ValueString())
	}
	if !plan.IdpSsoURL.IsNull() && !plan.IdpSsoURL.IsUnknown() {
		params.IdpSsoURL = clerkgo.String(plan.IdpSsoURL.ValueString())
	}
	if !plan.IdpCertificate.IsNull() && !plan.IdpCertificate.IsUnknown() {
		params.IdpCertificate = clerkgo.String(plan.IdpCertificate.ValueString())
	}
	if !plan.IdpMetadataURL.IsNull() && !plan.IdpMetadataURL.IsUnknown() {
		params.IdpMetadataURL = clerkgo.String(plan.IdpMetadataURL.ValueString())
	}
	if !plan.IdpMetadata.IsNull() && !plan.IdpMetadata.IsUnknown() {
		params.IdpMetadata = clerkgo.String(plan.IdpMetadata.ValueString())
	}

	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		params.Active = clerkgo.Bool(plan.Active.ValueBool())
	}
	if !plan.SyncUserAttributes.IsNull() && !plan.SyncUserAttributes.IsUnknown() {
		params.SyncUserAttributes = clerkgo.Bool(plan.SyncUserAttributes.ValueBool())
	}
	if !plan.AllowSubdomains.IsNull() && !plan.AllowSubdomains.IsUnknown() {
		params.AllowSubdomains = clerkgo.Bool(plan.AllowSubdomains.ValueBool())
	}
	if !plan.AllowIdpInitiated.IsNull() && !plan.AllowIdpInitiated.IsUnknown() {
		params.AllowIdpInitiated = clerkgo.Bool(plan.AllowIdpInitiated.ValueBool())
	}
	if !plan.DisableAdditionalIdentifications.IsNull() && !plan.DisableAdditionalIdentifications.IsUnknown() {
		params.DisableAdditionalIdentifications = clerkgo.Bool(plan.DisableAdditionalIdentifications.ValueBool())
	}

	if plan.AttributeMapping != nil {
		mapping := &samlconnection.AttributeMappingParams{}
		if !plan.AttributeMapping.UserID.IsNull() && !plan.AttributeMapping.UserID.IsUnknown() {
			mapping.UserID = plan.AttributeMapping.UserID.ValueString()
		}
		if !plan.AttributeMapping.EmailAddress.IsNull() && !plan.AttributeMapping.EmailAddress.IsUnknown() {
			mapping.EmailAddress = plan.AttributeMapping.EmailAddress.ValueString()
		}
		if !plan.AttributeMapping.FirstName.IsNull() && !plan.AttributeMapping.FirstName.IsUnknown() {
			mapping.FirstName = plan.AttributeMapping.FirstName.ValueString()
		}
		if !plan.AttributeMapping.LastName.IsNull() && !plan.AttributeMapping.LastName.IsUnknown() {
			mapping.LastName = plan.AttributeMapping.LastName.ValueString()
		}
		params.AttributeMapping = mapping
	}

	conn, err := samlconnection.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update SAML connection", err.Error())
		return
	}

	mapSAMLConnectionResponseToModel(conn, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated SAML connection", map[string]any{"id": conn.ID})
}

func (r *SAMLConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SAMLConnectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := samlconnection.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete SAML connection",
			fmt.Sprintf("Could not delete SAML connection ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted SAML connection", map[string]any{"id": state.ID.ValueString()})
}

func (r *SAMLConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapSAMLConnectionResponseToModel(conn *clerkgo.SAMLConnection, model *SAMLConnectionResourceModel) {
	model.ID = types.StringValue(conn.ID)
	model.Name = types.StringValue(conn.Name)
	model.Domain = types.StringValue(conn.Domain)
	model.SamlProvider = types.StringValue(conn.Provider)
	model.Active = types.BoolValue(conn.Active)
	model.SyncUserAttributes = types.BoolValue(conn.SyncUserAttributes)
	model.AllowSubdomains = types.BoolValue(conn.AllowSubdomains)
	model.AllowIdpInitiated = types.BoolValue(conn.AllowIdpInitiated)
	model.DisableAdditionalIdentifications = types.BoolValue(conn.DisableAdditionalIdentifications)
	model.AcsURL = types.StringValue(conn.AcsURL)
	model.SPEntityID = types.StringValue(conn.SPEntityID)
	model.SPMetadataURL = types.StringValue(conn.SPMetadataURL)
	model.UserCount = types.Int64Value(conn.UserCount)
	model.CreatedAt = types.StringValue(time.UnixMilli(conn.CreatedAt).UTC().Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(time.UnixMilli(conn.UpdatedAt).UTC().Format(time.RFC3339))

	if conn.OrganizationID != nil && *conn.OrganizationID != "" {
		model.OrganizationID = types.StringValue(*conn.OrganizationID)
	}
	if conn.IdpEntityID != nil && *conn.IdpEntityID != "" {
		model.IdpEntityID = types.StringValue(*conn.IdpEntityID)
	}
	if conn.IdpSsoURL != nil && *conn.IdpSsoURL != "" {
		model.IdpSsoURL = types.StringValue(*conn.IdpSsoURL)
	}
	if conn.IdpCertificate != nil && *conn.IdpCertificate != "" {
		model.IdpCertificate = types.StringValue(*conn.IdpCertificate)
	}
	if conn.IdpMetadataURL != nil && *conn.IdpMetadataURL != "" {
		model.IdpMetadataURL = types.StringValue(*conn.IdpMetadataURL)
	}
	if conn.IdpMetadata != nil && *conn.IdpMetadata != "" {
		model.IdpMetadata = types.StringValue(*conn.IdpMetadata)
	}

	model.AttributeMapping = &SAMLConnectionAttributeMappingModel{
		UserID:       types.StringValue(conn.AttributeMapping.UserID),
		EmailAddress: types.StringValue(conn.AttributeMapping.EmailAddress),
		FirstName:    types.StringValue(conn.AttributeMapping.FirstName),
		LastName:     types.StringValue(conn.AttributeMapping.LastName),
	}
}

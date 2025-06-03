// Copyright (c) KRUKON s.r.o

package provider

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/net/context"
	"io"
	"net/http"
	"net/http/httputil"
)

type RelationACSTFModel struct {
	ID           types.String `tfsdk:"id"`
	AccessCardID types.String `tfsdk:"access_card_id"`
	SystemID     types.String `tfsdk:"system_id"`
}

func NewRelationACSResource() resource.Resource {
	return &RelationACSResource{
		client: &http.Client{},
		token:  "",
	}
}

type RelationACSResource struct {
	client *http.Client
	token  string
}

func (r *RelationACSResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true},
			"access_card_id": schema.StringAttribute{Required: true},
			"system_id":      schema.StringAttribute{Required: true},
		},
	}
}

func (r *RelationACSResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data, ok := req.ProviderData.(goodAccessProviderModel)
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	if !ok || data.Token.IsNull() {
		resp.Diagnostics.AddError("Configuration Error", "Failed to retrieve provider data model.")
		return
	}

	r.token = data.Token.ValueString()
}

func (r *RelationACSResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "goodaccess_relation_ac_s"
}

func (r *RelationACSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RelationACSTFModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := fmt.Sprintf("https://integration.goodaccess.com/api/v1/relation/access-card/%s/system/%s",
		data.AccessCardID.ValueString(), data.SystemID.ValueString())

	httpReq, err := http.NewRequest("POST", url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Request Error", fmt.Sprintf("Failed to create POST request: %s", err))
		return
	}
	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Accept", "*/*")

	dump, err := httputil.DumpRequestOut(httpReq, true)
	if err != nil {
		fmt.Printf("DEBUG: Could not dump request: %s\n", err)
	} else {
		fmt.Printf("DEBUG: Outgoing HTTP request:\n%s\n", dump)
	}

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Network Error", fmt.Sprintf("POST request failed: %s", err))
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Create failed: %d: %s", httpResp.StatusCode, bodyBytes))
		return
	}

	// Synthetic ID: access_card_id + system_id
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.AccessCardID.ValueString(), data.SystemID.ValueString()))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *RelationACSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RelationACSTFModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpReq, _ := http.NewRequest("GET", "https://integration.goodaccess.com/api/v1/relations", nil)
	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Accept", "*/*")

	httpResp, err := r.client.Do(httpReq)
	if err != nil || httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("API Error", "Failed to fetch relations list")
		return
	}
	defer httpResp.Body.Close()

	bodyBytes, _ := io.ReadAll(httpResp.Body)
	var results []struct {
		ID           string `json:"id"`
		AccessCardID string `json:"accessCardId"`
		SystemID     string `json:"systemId"`
	}
	_ = json.Unmarshal(bodyBytes, &results)

	found := false
	for _, rel := range results {
		if rel.AccessCardID == state.AccessCardID.ValueString() && rel.SystemID == state.SystemID.ValueString() {
			state.ID = types.StringValue(fmt.Sprintf("%s:%s", rel.AccessCardID, rel.SystemID))
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	_ = resp.State.Set(ctx, &state)
}

func (r *RelationACSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RelationACSTFModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch relation ID first
	httpReq, _ := http.NewRequest("GET", "https://integration.goodaccess.com/api/v1/relations", nil)
	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Accept", "*/*")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("API Error", "Failed to lookup relation")
		return
	}
	defer httpResp.Body.Close()

	bodyBytes, _ := io.ReadAll(httpResp.Body)
	var relations []struct {
		ID           string `json:"id"`
		AccessCardID string `json:"accessCardId"`
		SystemID     string `json:"systemId"`
	}
	_ = json.Unmarshal(bodyBytes, &relations)

	var relationID string
	for _, rel := range relations {
		if rel.AccessCardID == state.AccessCardID.ValueString() && rel.SystemID == state.SystemID.ValueString() {
			relationID = rel.ID
			break
		}
	}

	if relationID == "" {
		resp.Diagnostics.AddWarning("Not Found", "Relation not found in GoodAccess — assuming deleted.")
		return
	}

	// Send DELETE
	url := fmt.Sprintf("https://integration.goodaccess.com/api/v1/relation/%s", relationID)
	httpReq, _ = http.NewRequest("DELETE", url, nil)
	httpReq.Header.Add("Authorization", "Bearer "+r.token)

	httpResp, err = r.client.Do(httpReq)
	if err != nil || httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("API Error", "Failed to delete relation")
		return
	}
}

func (r *RelationACSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Relations are immutable in GoodAccess API.
	// Instead of updating, Terraform should replace the resource.

	resp.Diagnostics.AddWarning(
		"Update Not Supported",
		"GoodAccess access-card↔system relations cannot be updated. Terraform will recreate the resource if changes are made.",
	)
}

// Copyright (c) KRUKON s.r.o

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io"
	"net/http"
)

type AccessCardModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func NewAccessCardResource() resource.Resource {
	return &AccessCardResource{
		client: &http.Client{},
		token:  "",
	}
}

type AccessCardResource struct {
	client *http.Client
	token  string
}

func (r *AccessCardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AccessCardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *AccessCardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "goodaccess_access_card"
}

func (r *AccessCardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccessCardModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := map[string]string{
		"name":        data.Name.ValueString(),
		"description": data.Description.ValueString(),
	}
	body, _ := json.Marshal(payload)

	httpReq, _ := http.NewRequest("POST", "https://integration.goodaccess.com/api/v1/access-card", bytes.NewBuffer(body))
	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil || httpResp.StatusCode != 200 {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("POST failed: %v", err))
		return
	}
	defer httpResp.Body.Close()

	var result struct {
		CreatedID string `json:"created_id"`
	}
	json.NewDecoder(httpResp.Body).Decode(&result)

	data.ID = types.StringValue(result.CreatedID)
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
func (r *AccessCardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccessCardModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read access card: ID is empty.")
		return
	}

	url := fmt.Sprintf("https://integration.goodaccess.com/api/v1/access-card/%s", id)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Request Error", fmt.Sprintf("Failed to create GET request: %s", err))
		return
	}
	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Accept", "*/*")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Network Error", fmt.Sprintf("Failed to send GET request: %s", err))
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		// Access card no longer exists — remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("GET failed with status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}

	// Parse response
	var result struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	bodyBytes, _ := io.ReadAll(httpResp.Body)
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Failed to decode response: %s", err))
		return
	}

	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Description = types.StringValue(result.Description)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *AccessCardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccessCardModel
	var state AccessCardModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot update access card: ID is empty.")
		return
	}

	url := fmt.Sprintf("https://integration.goodaccess.com/api/v1/access-card/%s", id)

	payload := map[string]string{
		"name":        plan.Name.ValueString(),
		"description": plan.Description.ValueString(),
	}
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		resp.Diagnostics.AddError("Request Error", fmt.Sprintf("Failed to create PUT request: %s", err))
		return
	}
	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Network Error", fmt.Sprintf("PUT request failed: %s", err))
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Update failed with status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}

	// Update state
	plan.ID = state.ID // preserve ID
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *AccessCardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccessCardModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete access card: ID is empty.")
		return
	}

	url := fmt.Sprintf("https://integration.goodaccess.com/api/v1/access-card/%s", id)

	httpReq, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Request Error", fmt.Sprintf("Failed to create DELETE request: %s", err))
		return
	}
	httpReq.Header.Add("Authorization", "Bearer "+r.token)

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Network Error", fmt.Sprintf("Failed to send DELETE request: %s", err))
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Delete failed with status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}
}

// Copyright (c) HashiCorp, Inc.

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
	"net/http/httputil"
)

type SystemResource struct {
	client *http.Client
	token  string
}

func NewSystemResource() resource.Resource {
	return &SystemResource{
		client: &http.Client{},
		token:  "",
	}
}

func (r *SystemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "goodaccess_system"
}

func (r *SystemResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SystemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name":     schema.StringAttribute{Required: true},
			"host":     schema.StringAttribute{Required: true},
			"uri":      schema.StringAttribute{Required: true},
			"port":     schema.StringAttribute{Required: true},
			"protocol": schema.StringAttribute{Required: true},
			"id":       schema.StringAttribute{Computed: true},
		},
	}
}

func (r *SystemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SystemModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := map[string]string{
		"name":     data.Name.ValueString(),
		"host":     data.Host.ValueString(),
		"uri":      data.Uri.ValueString(),
		"port":     data.Port.ValueString(),
		"protocol": data.Protocol.ValueString(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError("JSON Error", fmt.Sprintf("Could not marshal request body: %s", err))
		return
	}

	httpReq, err := http.NewRequest("POST", "https://integration.goodaccess.com/api/v1/system", bytes.NewBuffer(body))
	if err != nil {
		resp.Diagnostics.AddError("Request Error", fmt.Sprintf("Could not create request: %s", err))
		return
	}
	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Content-Type", "application/json")

	dump, err := httputil.DumpRequestOut(httpReq, true)
	if err != nil {
		fmt.Printf("DEBUG: Could not dump request: %s\n", err)
	} else {
		fmt.Printf("DEBUG: Outgoing HTTP request:\n%s\n", dump)
	}

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Network Error", fmt.Sprintf("Could not send request: %s", err))
		return
	}
	defer httpResp.Body.Close()

	bodyBytes, _ := io.ReadAll(httpResp.Body)
	fmt.Printf("DEBUG: response: %s\n", bodyBytes)
	if httpResp.StatusCode != http.StatusOK {
		// Try to extract API error message
		var errResp map[string]interface{}
		_ = json.Unmarshal(bodyBytes, &errResp)

		msg := fmt.Sprintf("HTTP %d: %s", httpResp.StatusCode, string(bodyBytes))
		if errMsg, ok := errResp["error_description"].(string); ok {
			msg = errMsg
		}

		resp.Diagnostics.AddError("API Error", msg)
		return
	}

	// Success case
	var result struct {
		CreatedID string `json:"created_id"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Could not parse success response: %s", err))
		return
	}

	data.ID = types.StringValue(result.CreatedID)
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *SystemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SystemModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete system: ID is empty.")
		return
	}

	url := fmt.Sprintf("https://integration.goodaccess.com/api/v1/system/%s", id)

	httpReq, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Request Error", fmt.Sprintf("Failed to create DELETE request: %s", err))
		return
	}

	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Accept", "*/*")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Network Error", fmt.Sprintf("Request failed: %s", err))
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("DELETE request failed with status %d. Body: %s", httpResp.StatusCode, string(bodyBytes)),
		)
		return
	}
}

func (r *SystemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SystemModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read system: ID is empty.")
		return
	}

	// Fetch all systems
	url := "https://integration.goodaccess.com/api/v1/systems"
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Request Error", fmt.Sprintf("Failed to create request: %s", err))
		return
	}
	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Accept", "*/*")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Network Error", fmt.Sprintf("GET request failed: %s", err))
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to read system list (status %d): %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}

	// Parse the response
	var systems []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Host     string `json:"host"`
		Uri      string `json:"uri"`
		Port     string `json:"port"`
		Protocol string `json:"protocol"` // optional, API may or may not send it
	}
	bodyBytes, _ := io.ReadAll(httpResp.Body)
	if err := json.Unmarshal(bodyBytes, &systems); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Failed to parse response: %s", err))
		return
	}

	// Find the system by ID
	var found bool
	for _, s := range systems {
		if s.ID == id {
			state.ID = types.StringValue(s.ID)
			state.Name = types.StringValue(s.Name)
			state.Host = types.StringValue(s.Host)
			state.Uri = types.StringValue(s.Uri)
			state.Port = types.StringValue(s.Port)
			state.Protocol = types.StringValue(s.Protocol) // optional: handle "" if needed
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *SystemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SystemModel
	var state SystemModel

	// Get the new desired values
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	// Get the current state (needed for the ID)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot update system: ID is empty.")
		return
	}

	url := fmt.Sprintf("https://integration.goodaccess.com/api/v1/system/%s", id)

	// Construct request payload from the planned state
	payload := map[string]string{
		"name":     plan.Name.ValueString(),
		"host":     plan.Host.ValueString(),
		"uri":      plan.Uri.ValueString(),
		"port":     plan.Port.ValueString(),
		"protocol": plan.Protocol.ValueString(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError("JSON Error", fmt.Sprintf("Failed to encode update payload: %s", err))
		return
	}

	// Prepare the HTTP PUT request
	httpReq, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		resp.Diagnostics.AddError("Request Error", fmt.Sprintf("Failed to create update request: %s", err))
		return
	}
	httpReq.Header.Add("Authorization", "Bearer "+r.token)
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("Accept", "*/*")

	// Send the request
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Network Error", fmt.Sprintf("Update request failed: %s", err))
		return
	}
	defer httpResp.Body.Close()

	// Check API response
	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Update failed with status %d: %s", httpResp.StatusCode, string(bodyBytes)),
		)
		return
	}

	// Save the updated state
	plan.ID = state.ID // preserve ID in state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

type SystemModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Host     types.String `tfsdk:"host"`
	Uri      types.String `tfsdk:"uri"`
	Port     types.String `tfsdk:"port"`
	Protocol types.String `tfsdk:"protocol"`
}

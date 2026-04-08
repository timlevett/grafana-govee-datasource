package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/govee-datasource/pkg/govee"
	"github.com/grafana/govee-datasource/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ backend.CallResourceHandler   = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	config, err := models.LoadPluginSettings(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	client := govee.NewClient(config.Secrets.ApiKey)
	
	return &Datasource{
		client: client,
	}, nil
}

// Datasource is a datasource which can respond to data queries, reports
// its health and interacts with the Govee API.
type Datasource struct {
	client *govee.Client
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	QueryType string `json:"queryType"`
	DeviceID  string `json:"deviceId"`
	Model     string `json:"model"`
}

func (d *Datasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	// Handle different query types
	switch qm.QueryType {
	case "devices":
		return d.queryDevices(ctx, query)
	case "deviceState":
		if qm.DeviceID == "" || qm.Model == "" {
			return backend.ErrDataResponse(backend.StatusBadRequest, "deviceId and model are required for deviceState query")
		}
		return d.queryDeviceState(ctx, qm.DeviceID, qm.Model, query)
	default:
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("unknown query type: %s", qm.QueryType))
	}
}

func (d *Datasource) queryDevices(ctx context.Context, query backend.DataQuery) backend.DataResponse {
	devices, err := d.client.GetDevices(ctx)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("failed to get devices: %v", err))
	}

	// Create a table frame with device information
	frame := data.NewFrame("devices",
		data.NewField("deviceId", nil, []string{}),
		data.NewField("deviceName", nil, []string{}),
		data.NewField("model", nil, []string{}),
		data.NewField("controllable", nil, []bool{}),
		data.NewField("retrievable", nil, []bool{}),
	)

	for _, device := range devices {
		frame.AppendRow(
			device.DeviceID,
			device.DeviceName,
			device.Model,
			device.Controllable,
			device.Retrievable,
		)
	}

	return backend.DataResponse{
		Frames: []*data.Frame{frame},
	}
}

func (d *Datasource) queryDeviceState(ctx context.Context, deviceID, model string, query backend.DataQuery) backend.DataResponse {
	state, err := d.client.GetDeviceState(ctx, deviceID, model)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("failed to get device state: %v", err))
	}

	// Create a time series frame
	now := time.Now()
	frame := data.NewFrame("deviceState",
		data.NewField("time", nil, []time.Time{now}),
	)

	// Add properties as fields
	for _, prop := range state.Properties {
		for key, value := range prop {
			// Convert value to appropriate type
			switch v := value.(type) {
			case float64:
				frame.Fields = append(frame.Fields, data.NewField(key, nil, []float64{v}))
			case int:
				frame.Fields = append(frame.Fields, data.NewField(key, nil, []int64{int64(v)}))
			case bool:
				frame.Fields = append(frame.Fields, data.NewField(key, nil, []bool{v}))
			case string:
				frame.Fields = append(frame.Fields, data.NewField(key, nil, []string{v}))
			default:
				// For unknown types, convert to string
				frame.Fields = append(frame.Fields, data.NewField(key, nil, []string{fmt.Sprintf("%v", v)}))
			}
		}
	}

	return backend.DataResponse{
		Frames: []*data.Frame{frame},
	}
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)

	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Unable to load settings",
		}, nil
	}

	if config.Secrets.ApiKey == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "API key is missing",
		}, nil
	}

	// Test the API connection by trying to get devices
	_, err = d.client.GetDevices(ctx)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Failed to connect to Govee API: %v", err),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Successfully connected to Govee API",
	}, nil
}

// CallResource handles resource calls from the frontend.
// This is used for template variable queries.
func (d *Datasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	// Handle different resource paths
	switch req.Path {
	case "devices":
		return d.handleDevicesResource(ctx, sender)
	default:
		return sender.Send(&backend.CallResourceResponse{
			Status:  http.StatusNotFound,
			Headers: make(map[string][]string),
			Body:    []byte(fmt.Sprintf(`{"error": "unknown resource path: %s"}`, req.Path)),
		})
	}
}

func (d *Datasource) handleDevicesResource(ctx context.Context, sender backend.CallResourceResponseSender) error {
	devices, err := d.client.GetDevices(ctx)
	if err != nil {
		backend.Logger.Error("Failed to get devices for resource call", "error", err)
		return sender.Send(&backend.CallResourceResponse{
			Status:  http.StatusInternalServerError,
			Headers: make(map[string][]string),
			Body:    []byte(fmt.Sprintf(`{"error": "failed to get devices: %v"}`, err)),
		})
	}

	backend.Logger.Info("Fetched devices for resource call", "count", len(devices))

	// Format devices for Grafana template variables
	// Each variable option needs text and value
	type VariableOption struct {
		Text  string `json:"text"`
		Value string `json:"value"`
	}

	options := make([]VariableOption, 0, len(devices))
	for _, device := range devices {
		// Use device name as text, device ID as value
		options = append(options, VariableOption{
			Text:  device.DeviceName,
			Value: device.DeviceID,
		})
	}

	// Also add options for device ID + model combination (useful for deviceState queries)
	// Format: "deviceId|model"
	combinedOptions := make([]VariableOption, 0, len(devices))
	for _, device := range devices {
		combinedOptions = append(combinedOptions, VariableOption{
			Text:  fmt.Sprintf("%s (%s)", device.DeviceName, device.Model),
			Value: fmt.Sprintf("%s|%s", device.DeviceID, device.Model),
		})
	}

	// Create response with both formats
	response := map[string]interface{}{
		"devices":        options,
		"devicesWithModel": combinedOptions,
	}

	body, err := json.Marshal(response)
	if err != nil {
		return sender.Send(&backend.CallResourceResponse{
			Status:  http.StatusInternalServerError,
			Headers: make(map[string][]string),
			Body:    []byte(fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err)),
		})
	}

	headers := make(map[string][]string)
	headers["Content-Type"] = []string{"application/json"}

	return sender.Send(&backend.CallResourceResponse{
		Status:  http.StatusOK,
		Headers: headers,
		Body:    body,
	})
}

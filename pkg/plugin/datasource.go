package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/timlevett/grafana-govee-datasource/pkg/models"
)

// Verify interface compliance at compile time.
var (
	_ backend.QueryDataHandler      = (*GoveeDatasource)(nil)
	_ backend.CheckHealthHandler    = (*GoveeDatasource)(nil)
	_ backend.CallResourceHandler   = (*GoveeDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*GoveeDatasource)(nil)
)

// ---------------------------------------------------------------------------
// GoveeDatasource — the plugin instance
// ---------------------------------------------------------------------------

// GoveeDatasource is created once per datasource instance (per saved datasource
// configuration in Grafana). It holds a GoveeClient that can be shared across
// requests because the API key is fetched fresh from settings each call.
type GoveeDatasource struct {
	settings backend.DataSourceInstanceSettings
	client   *GoveeClient
}

// NewGoveeDatasource is called by the instance manager when a new datasource
// instance is created or its settings change.
func NewGoveeDatasource(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	ps, err := models.LoadPluginSettings(settings)
	if err != nil {
		return nil, fmt.Errorf("govee datasource: load settings: %w", err)
	}

	return &GoveeDatasource{
		settings: settings,
		client:   NewGoveeClient(ps.APIBaseURL),
	}, nil
}

// Dispose is called when the datasource instance is removed or settings change.
// Nothing to clean up here, but the interface must be satisfied.
func (d *GoveeDatasource) Dispose() {}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (d *GoveeDatasource) apiKey() string {
	return models.APIKey(d.settings)
}

// ---------------------------------------------------------------------------
// CheckHealth — validates the API key by listing devices
// ---------------------------------------------------------------------------

func (d *GoveeDatasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	apiKey := d.apiKey()
	if apiKey == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "API key is not configured. Add your Govee API key in the datasource settings.",
		}, nil
	}

	devices, err := d.client.ListDevices(ctx, apiKey)
	if err != nil {
		log.DefaultLogger.Error("CheckHealth: ListDevices failed", "error", err.Error())
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Could not connect to Govee API: %s", err.Error()),
		}, nil
	}

	msg := fmt.Sprintf(
		"Successfully connected to Govee API. Found %d device(s). Rate limit remaining today: %d/10,000.",
		len(devices),
		d.client.RateLimitRemaining(),
	)

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: msg,
	}, nil
}

// ---------------------------------------------------------------------------
// CallResource — handles frontend resource requests (e.g. /devices)
// ---------------------------------------------------------------------------

func (d *GoveeDatasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	switch req.Path {
	case "devices":
		return d.handleGetDevices(ctx, sender)
	default:
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusNotFound,
			Body:   []byte(`{"error":"unknown resource path"}`),
		})
	}
}

type deviceListResourceResponse struct {
	Devices []Device `json:"devices"`
}

func (d *GoveeDatasource) handleGetDevices(ctx context.Context, sender backend.CallResourceResponseSender) error {
	apiKey := d.apiKey()
	if apiKey == "" {
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusUnauthorized,
			Body:   []byte(`{"error":"API key not configured"}`),
		})
	}

	devices, err := d.client.ListDevices(ctx, apiKey)
	if err != nil {
		log.DefaultLogger.Error("CallResource /devices: ListDevices failed", "error", err.Error())
		body, _ := json.Marshal(map[string]string{"error": err.Error()})
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusBadGateway,
			Body:   body,
		})
	}

	body, err := json.Marshal(deviceListResourceResponse{Devices: devices})
	if err != nil {
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusInternalServerError,
			Body:   []byte(`{"error":"internal marshal error"}`),
		})
	}

	return sender.Send(&backend.CallResourceResponse{
		Status:  http.StatusOK,
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    body,
	})
}

// ---------------------------------------------------------------------------
// QueryData — handles panel queries
// ---------------------------------------------------------------------------

func (d *GoveeDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	apiKey := d.apiKey()
	if apiKey == "" {
		for _, q := range req.Queries {
			response.Responses[q.RefID] = backend.DataResponse{
				Error: fmt.Errorf("API key not configured"),
			}
		}
		return response, nil
	}

	for _, q := range req.Queries {
		res := d.runQuery(ctx, apiKey, q)
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *GoveeDatasource) runQuery(ctx context.Context, apiKey string, q backend.DataQuery) backend.DataResponse {
	qm, err := models.ParseQueryModel(q.JSON)
	if err != nil {
		return backend.DataResponse{Error: fmt.Errorf("parse query: %w", err)}
	}

	if qm.DeviceID == "" || qm.SKU == "" {
		return backend.DataResponse{Error: fmt.Errorf("device and SKU must be selected")}
	}

	instance := qm.EffectiveInstance()
	if instance == "" {
		return backend.DataResponse{Error: fmt.Errorf("metric/instance must be specified")}
	}

	stateData, err := d.client.QueryDeviceState(ctx, apiKey, qm.SKU, qm.DeviceID)
	if err != nil {
		log.DefaultLogger.Error("QueryData: QueryDeviceState failed",
			"device", qm.DeviceID, "sku", qm.SKU, "error", err.Error())
		return backend.DataResponse{Error: fmt.Errorf("query device state: %w", err)}
	}

	// Find the requested capability in the state response.
	var capValue interface{}
	found := false
	for _, cap := range stateData.Capabilities {
		if cap.Instance == instance {
			capValue = cap.State
			found = true
			break
		}
	}

	if !found {
		return backend.DataResponse{
			Error: fmt.Errorf("capability %q not found in device state (device has %d capabilities)", instance, len(stateData.Capabilities)),
		}
	}

	// Convert the raw state value to a float64 for Grafana frames.
	numericValue, err := toFloat64(capValue)
	if err != nil {
		// Return as a string frame instead.
		return d.buildStringFrame(q, qm, instance, fmt.Sprintf("%v", capValue))
	}

	return d.buildNumericFrame(q, qm, instance, numericValue)
}

// ---------------------------------------------------------------------------
// Data frame builders
// ---------------------------------------------------------------------------

// buildNumericFrame creates a data frame with a single numeric time-series
// point at the query's "now" time. For a true time series you would need
// Govee to provide historical data — the API currently only returns current
// state, so we emit a single point at query time. This is appropriate for
// stat / gauge panels and current-state displays.
func (d *GoveeDatasource) buildNumericFrame(
	q backend.DataQuery,
	qm *models.QueryModel,
	instance string,
	value float64,
) backend.DataResponse {
	now := time.Now()

	frameLabel := qm.DeviceName
	if frameLabel == "" {
		frameLabel = qm.DeviceID
	}
	frameName := fmt.Sprintf("%s — %s", frameLabel, instance)

	timeField := data.NewField("time", nil, []time.Time{now})
	valueField := data.NewField("value", data.Labels{"device": frameLabel, "metric": instance}, []float64{value})
	valueField.Config = &data.FieldConfig{
		DisplayNameFromDS: frameName,
	}

	frame := data.NewFrame(frameName, timeField, valueField)

	if qm.QueryType == models.QueryTypeTimeSeries {
		frame.SetMeta(&data.FrameMeta{
			Type: data.FrameTypeTimeSeriesMany,
		})
	}

	return backend.DataResponse{Frames: []*data.Frame{frame}}
}

// buildStringFrame returns a single-row string frame for non-numeric state
// values (e.g. powerState = "on").
func (d *GoveeDatasource) buildStringFrame(
	q backend.DataQuery,
	qm *models.QueryModel,
	instance string,
	value string,
) backend.DataResponse {
	now := time.Now()

	frameLabel := qm.DeviceName
	if frameLabel == "" {
		frameLabel = qm.DeviceID
	}
	frameName := fmt.Sprintf("%s — %s", frameLabel, instance)

	timeField := data.NewField("time", nil, []time.Time{now})
	valueField := data.NewField("value", data.Labels{"device": frameLabel, "metric": instance}, []string{value})
	valueField.Config = &data.FieldConfig{
		DisplayNameFromDS: frameName,
	}

	frame := data.NewFrame(frameName, timeField, valueField)

	return backend.DataResponse{Frames: []*data.Frame{frame}}
}

// ---------------------------------------------------------------------------
// Type conversion helpers
// ---------------------------------------------------------------------------

// toFloat64 attempts to coerce a capability state value to float64.
// Govee returns numbers as json.Number or float64 after unmarshalling into
// interface{}, so we handle both. Boolean values (powerState) are mapped to
// 1.0/0.0. Returns an error for opaque string values.
func toFloat64(v interface{}) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return 0, fmt.Errorf("convert json.Number: %w", err)
		}
		return f, nil
	case bool:
		if t {
			return 1.0, nil
		}
		return 0.0, nil
	case int:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case map[string]interface{}:
		// Some Govee states are nested objects like {"value": 42, "unit": "℃"}.
		if val, ok := t["value"]; ok {
			return toFloat64(val)
		}
		return 0, fmt.Errorf("complex object state with no 'value' key")
	default:
		return 0, fmt.Errorf("non-numeric state type %T", v)
	}
}

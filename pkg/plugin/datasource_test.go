package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/timlevett/grafana-govee-datasource/pkg/models"
)

// ---------------------------------------------------------------------------
// toFloat64
// ---------------------------------------------------------------------------

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    float64
		wantErr bool
	}{
		{"float64", float64(3.14), 3.14, false},
		{"json.Number integer", json.Number("42"), 42.0, false},
		{"json.Number float", json.Number("1.5"), 1.5, false},
		{"json.Number invalid", json.Number("not-a-number"), 0, true},
		{"bool true", true, 1.0, false},
		{"bool false", false, 0.0, false},
		{"int", int(7), 7.0, false},
		{"int64", int64(1000), 1000.0, false},
		{"map with value key", map[string]interface{}{"value": float64(25)}, 25.0, false},
		{"map nested json.Number", map[string]interface{}{"value": json.Number("99")}, 99.0, false},
		{"map without value key", map[string]interface{}{"unit": "℃"}, 0, true},
		{"string", "on", 0, true},
		{"nil", nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toFloat64(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toFloat64(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("toFloat64(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CheckHealth
// ---------------------------------------------------------------------------

func makeDS(settings backend.DataSourceInstanceSettings, serverURL string) *GoveeDatasource {
	return &GoveeDatasource{
		settings: settings,
		client:   NewGoveeClient(serverURL),
	}
}

func TestCheckHealth_MissingAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when API key is missing")
	}))
	defer srv.Close()

	ds := makeDS(backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{},
	}, srv.URL)

	result, err := ds.CheckHealth(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != backend.HealthStatusError {
		t.Errorf("status: got %v want Error", result.Status)
	}
}

func TestCheckHealth_Success(t *testing.T) {
	body, _ := json.Marshal(DeviceListPayload{
		Code:    200,
		Message: "success",
		Data: []Device{
			{SKU: "H5101", Device: "AA:BB:CC", DeviceName: "Sensor"},
		},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	ds := makeDS(backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{"apiKey": "valid-key"},
	}, srv.URL)

	result, err := ds.CheckHealth(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != backend.HealthStatusOk {
		t.Errorf("status: got %v want Ok; message: %s", result.Status, result.Message)
	}
}

func TestCheckHealth_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":403,"message":"Unauthorized"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	ds := makeDS(backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{"apiKey": "bad-key"},
	}, srv.URL)

	result, err := ds.CheckHealth(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != backend.HealthStatusError {
		t.Errorf("status: got %v want Error", result.Status)
	}
}

// ---------------------------------------------------------------------------
// QueryData
// ---------------------------------------------------------------------------

func queryJSON(t *testing.T, qm models.QueryModel) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(qm)
	if err != nil {
		t.Fatalf("marshal query model: %v", err)
	}
	return b
}

func TestQueryData_MissingAPIKey(t *testing.T) {
	ds := makeDS(backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{},
	}, "http://localhost:1")

	req := &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "A", JSON: queryJSON(t, models.QueryModel{DeviceID: "AA:BB", SKU: "H5101", Metric: "temperature"})},
		},
	}
	resp, err := ds.QueryData(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.Responses["A"]
	if !ok || r.Error == nil {
		t.Error("expected error response for missing API key")
	}
}

func TestQueryData_MissingDeviceID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when device ID is missing")
	}))
	defer srv.Close()

	ds := makeDS(backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{"apiKey": "test-key"},
	}, srv.URL)

	req := &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "A", JSON: queryJSON(t, models.QueryModel{DeviceID: "", SKU: "", Metric: "temperature"})},
		},
	}
	resp, err := ds.QueryData(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.Responses["A"]
	if !ok || r.Error == nil {
		t.Error("expected error response for empty device ID")
	}
}

func TestQueryData_Success_NumericCapability(t *testing.T) {
	stateResp, _ := json.Marshal(DeviceStatePayload{
		Code: 200,
		Data: DeviceStateData{
			SKU:    "H5101",
			Device: "AA:BB:CC",
			Capabilities: []StateCapabilityValue{
				{Instance: "temperature", State: float64(22.5)},
			},
		},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(stateResp)
	}))
	defer srv.Close()

	ds := makeDS(backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{"apiKey": "test-key"},
	}, srv.URL)

	req := &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "A", JSON: queryJSON(t, models.QueryModel{
				DeviceID: "AA:BB:CC",
				SKU:      "H5101",
				Metric:   "temperature",
			})},
		},
	}
	resp, err := ds.QueryData(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := resp.Responses["A"]
	if r.Error != nil {
		t.Fatalf("unexpected response error: %v", r.Error)
	}
	if len(r.Frames) != 1 {
		t.Fatalf("frames: got %d want 1", len(r.Frames))
	}
}

func TestQueryData_CapabilityNotFound(t *testing.T) {
	stateResp, _ := json.Marshal(DeviceStatePayload{
		Code: 200,
		Data: DeviceStateData{
			SKU:    "H5101",
			Device: "AA:BB:CC",
			Capabilities: []StateCapabilityValue{
				{Instance: "battery", State: float64(80)},
			},
		},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(stateResp)
	}))
	defer srv.Close()

	ds := makeDS(backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{"apiKey": "test-key"},
	}, srv.URL)

	req := &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "A", JSON: queryJSON(t, models.QueryModel{
				DeviceID: "AA:BB:CC",
				SKU:      "H5101",
				Metric:   "temperature", // not in state response
			})},
		},
	}
	resp, err := ds.QueryData(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := resp.Responses["A"]
	if r.Error == nil {
		t.Error("expected error when capability not found")
	}
}

func TestQueryData_StringCapabilityBuildsStringFrame(t *testing.T) {
	stateResp, _ := json.Marshal(DeviceStatePayload{
		Code: 200,
		Data: DeviceStateData{
			SKU:    "H5101",
			Device: "AA:BB:CC",
			Capabilities: []StateCapabilityValue{
				{Instance: "powerState", State: "on"},
			},
		},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(stateResp)
	}))
	defer srv.Close()

	ds := makeDS(backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{"apiKey": "test-key"},
	}, srv.URL)

	req := &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "A", JSON: queryJSON(t, models.QueryModel{
				DeviceID: "AA:BB:CC",
				SKU:      "H5101",
				Metric:   "powerState",
			})},
		},
	}
	resp, err := ds.QueryData(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := resp.Responses["A"]
	if r.Error != nil {
		t.Fatalf("unexpected error: %v", r.Error)
	}
	if len(r.Frames) != 1 {
		t.Fatalf("frames: got %d want 1", len(r.Frames))
	}
}

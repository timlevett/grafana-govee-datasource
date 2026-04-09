package models_test

import (
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/timlevett/grafana-govee-datasource/pkg/models"
)

// ---------------------------------------------------------------------------
// ParseQueryModel
// ---------------------------------------------------------------------------

func TestParseQueryModel_ValidFull(t *testing.T) {
	raw := json.RawMessage(`{"queryType":"timeseries","deviceId":"AA:BB","sku":"H5101","metric":"temperature","deviceName":"My Sensor"}`)
	qm, err := models.ParseQueryModel(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qm.DeviceID != "AA:BB" {
		t.Errorf("DeviceID: got %q want AA:BB", qm.DeviceID)
	}
	if qm.SKU != "H5101" {
		t.Errorf("SKU: got %q want H5101", qm.SKU)
	}
	if qm.QueryType != models.QueryTypeTimeSeries {
		t.Errorf("QueryType: got %q want timeseries", qm.QueryType)
	}
	if qm.DeviceName != "My Sensor" {
		t.Errorf("DeviceName: got %q want My Sensor", qm.DeviceName)
	}
}

func TestParseQueryModel_DefaultsQueryType(t *testing.T) {
	raw := json.RawMessage(`{"deviceId":"AA:BB","sku":"H5101","metric":"temperature"}`)
	qm, err := models.ParseQueryModel(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qm.QueryType != models.QueryTypeCurrent {
		t.Errorf("QueryType: got %q want current", qm.QueryType)
	}
}

func TestParseQueryModel_EmptyJSON(t *testing.T) {
	raw := json.RawMessage(`{}`)
	qm, err := models.ParseQueryModel(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qm.QueryType != models.QueryTypeCurrent {
		t.Errorf("QueryType: got %q want current", qm.QueryType)
	}
	if qm.DeviceID != "" {
		t.Errorf("DeviceID should be empty, got %q", qm.DeviceID)
	}
}

func TestParseQueryModel_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`not valid json`)
	_, err := models.ParseQueryModel(raw)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseQueryModel_CustomInstance(t *testing.T) {
	raw := json.RawMessage(`{"metric":"custom","customInstance":"sensorTemperature"}`)
	qm, err := models.ParseQueryModel(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qm.CustomInstance != "sensorTemperature" {
		t.Errorf("CustomInstance: got %q want sensorTemperature", qm.CustomInstance)
	}
}

// ---------------------------------------------------------------------------
// QueryModel.EffectiveInstance
// ---------------------------------------------------------------------------

func TestEffectiveInstance_NonCustom(t *testing.T) {
	qm := &models.QueryModel{Metric: "temperature"}
	if got := qm.EffectiveInstance(); got != "temperature" {
		t.Errorf("got %q want temperature", got)
	}
}

func TestEffectiveInstance_CustomWithValue(t *testing.T) {
	qm := &models.QueryModel{Metric: "custom", CustomInstance: "sensorTemperature"}
	if got := qm.EffectiveInstance(); got != "sensorTemperature" {
		t.Errorf("got %q want sensorTemperature", got)
	}
}

func TestEffectiveInstance_CustomWithoutValue(t *testing.T) {
	// When custom is selected but no CustomInstance is provided, falls back to
	// returning "custom" (the Metric value itself), matching the EffectiveInstance
	// implementation which only substitutes CustomInstance when it is non-empty.
	qm := &models.QueryModel{Metric: "custom", CustomInstance: ""}
	if got := qm.EffectiveInstance(); got != "custom" {
		t.Errorf("got %q want \"custom\"", got)
	}
}

// ---------------------------------------------------------------------------
// APIKey
// ---------------------------------------------------------------------------

func TestAPIKey_Present(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{"apiKey": "my-secret-key"},
	}
	got := models.APIKey(settings)
	if got != "my-secret-key" {
		t.Errorf("got %q want my-secret-key", got)
	}
}

func TestAPIKey_Missing(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{
		DecryptedSecureJSONData: map[string]string{},
	}
	got := models.APIKey(settings)
	if got != "" {
		t.Errorf("got %q want empty string", got)
	}
}

func TestAPIKey_NilMap(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{}
	got := models.APIKey(settings)
	if got != "" {
		t.Errorf("got %q want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// LoadPluginSettings
// ---------------------------------------------------------------------------

func TestLoadPluginSettings_EmptyData(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{}
	ps, err := models.LoadPluginSettings(settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ps.APIBaseURL != "" {
		t.Errorf("APIBaseURL: got %q want empty", ps.APIBaseURL)
	}
}

func TestLoadPluginSettings_WithBaseURL(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{
		JSONData: json.RawMessage(`{"apiBaseUrl":"https://custom.example.com"}`),
	}
	ps, err := models.LoadPluginSettings(settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ps.APIBaseURL != "https://custom.example.com" {
		t.Errorf("APIBaseURL: got %q want https://custom.example.com", ps.APIBaseURL)
	}
}

func TestLoadPluginSettings_InvalidJSON(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{
		JSONData: json.RawMessage(`not valid json`),
	}
	_, err := models.LoadPluginSettings(settings)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

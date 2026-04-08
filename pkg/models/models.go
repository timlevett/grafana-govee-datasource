// Package models contains shared data models used by the Govee datasource backend.
package models

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// ---------------------------------------------------------------------------
// Datasource settings
// ---------------------------------------------------------------------------

// PluginSettings holds the non-sensitive datasource JSON data.
type PluginSettings struct {
	// APIBaseURL allows overriding the default Govee API base URL.
	// If empty, the plugin uses https://openapi.api.govee.com.
	APIBaseURL string `json:"apiBaseUrl,omitempty"`
}

// LoadPluginSettings deserialises the datasource JSON data from the supplied
// DataSourceInstanceSettings. Returns an error if the JSON is malformed.
func LoadPluginSettings(settings backend.DataSourceInstanceSettings) (*PluginSettings, error) {
	ps := &PluginSettings{}
	if len(settings.JSONData) > 0 {
		if err := json.Unmarshal(settings.JSONData, ps); err != nil {
			return nil, fmt.Errorf("parsing plugin settings: %w", err)
		}
	}
	return ps, nil
}

// APIKey retrieves the Govee API key from the secure JSON data map.
// Returns an empty string (not an error) if the key is absent — callers
// should treat an empty key as a configuration error.
func APIKey(settings backend.DataSourceInstanceSettings) string {
	return settings.DecryptedSecureJSONData["apiKey"]
}

// ---------------------------------------------------------------------------
// Query model (mirrors src/types.ts GoveeQuery)
// ---------------------------------------------------------------------------

// QueryType distinguishes between time-series and point-in-time queries.
type QueryType string

const (
	QueryTypeTimeSeries QueryType = "timeseries"
	QueryTypeCurrent    QueryType = "current"
)

// QueryModel is the parsed query sent from the frontend panel.
type QueryModel struct {
	// QueryType is "timeseries" or "current".
	QueryType QueryType `json:"queryType"`
	// DeviceID is the Govee device MAC address / device identifier.
	DeviceID string `json:"deviceId"`
	// SKU is the Govee model string required by the state API.
	SKU string `json:"sku"`
	// Metric is the capability instance name to surface (e.g. "temperature").
	Metric string `json:"metric"`
	// CustomInstance is used when Metric is "custom".
	CustomInstance string `json:"customInstance,omitempty"`
	// DeviceName is purely informational, used for frame labels.
	DeviceName string `json:"deviceName,omitempty"`
}

// EffectiveInstance returns the capability instance string that should be
// queried. If Metric is "custom" it returns CustomInstance; otherwise Metric.
func (q *QueryModel) EffectiveInstance() string {
	if q.Metric == "custom" && q.CustomInstance != "" {
		return q.CustomInstance
	}
	return q.Metric
}

// ParseQueryModel deserialises a raw query JSON blob into a QueryModel.
func ParseQueryModel(raw json.RawMessage) (*QueryModel, error) {
	qm := &QueryModel{}
	if err := json.Unmarshal(raw, qm); err != nil {
		return nil, fmt.Errorf("parsing query model: %w", err)
	}
	if qm.QueryType == "" {
		qm.QueryType = QueryTypeCurrent
	}
	return qm, nil
}

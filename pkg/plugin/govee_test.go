package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// rateLimiter
// ---------------------------------------------------------------------------

func TestRateLimiterAllow_IncrementsCounter(t *testing.T) {
	r := newRateLimiter()
	for i := 0; i < 5; i++ {
		if !r.Allow() {
			t.Fatalf("Allow() returned false on call %d; expected true", i+1)
		}
	}
	if r.count != 5 {
		t.Errorf("count: got %d want 5", r.count)
	}
}

func TestRateLimiterAllow_BlocksAtLimit(t *testing.T) {
	r := newRateLimiter()
	r.count = dailyRateLimit
	if r.Allow() {
		t.Error("Allow() returned true when limit already reached; expected false")
	}
}

func TestRateLimiterAllow_ResetsOnDayChange(t *testing.T) {
	r := newRateLimiter()
	r.count = dailyRateLimit
	r.resetDate = "2000-01-01" // force stale date
	if !r.Allow() {
		t.Error("Allow() returned false after day change; expected true (reset)")
	}
	if r.count != 1 {
		t.Errorf("count after reset: got %d want 1", r.count)
	}
}

func TestRateLimiterRemaining_Full(t *testing.T) {
	r := newRateLimiter()
	if got := r.Remaining(); got != dailyRateLimit {
		t.Errorf("Remaining: got %d want %d", got, dailyRateLimit)
	}
}

func TestRateLimiterRemaining_Decrements(t *testing.T) {
	r := newRateLimiter()
	r.Allow()
	r.Allow()
	if got := r.Remaining(); got != dailyRateLimit-2 {
		t.Errorf("Remaining: got %d want %d", got, dailyRateLimit-2)
	}
}

func TestRateLimiterRemaining_NeverNegative(t *testing.T) {
	r := newRateLimiter()
	r.count = dailyRateLimit + 100
	if got := r.Remaining(); got != 0 {
		t.Errorf("Remaining: got %d want 0 (should not go negative)", got)
	}
}

func TestRateLimiterRemaining_ResetsOnDayChange(t *testing.T) {
	r := newRateLimiter()
	r.count = 9000
	r.resetDate = "2000-01-01"
	if got := r.Remaining(); got != dailyRateLimit {
		t.Errorf("Remaining after day change: got %d want %d", got, dailyRateLimit)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestClient(t *testing.T, handler http.HandlerFunc) (*GoveeClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewGoveeClient(srv.URL), srv
}

func deviceListBody(t *testing.T, devices []Device) []byte {
	t.Helper()
	b, err := json.Marshal(DeviceListPayload{Code: 200, Message: "success", Data: devices})
	if err != nil {
		t.Fatalf("marshal device list: %v", err)
	}
	return b
}

func stateBody(t *testing.T, caps []StateCapabilityValue) []byte {
	t.Helper()
	b, err := json.Marshal(DeviceStatePayload{
		Code:    200,
		Message: "success",
		Data: DeviceStateData{
			SKU:          "H5101",
			Device:       "AA:BB:CC",
			Capabilities: caps,
		},
	})
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	return b
}

// ---------------------------------------------------------------------------
// ListDevices
// ---------------------------------------------------------------------------

func TestListDevices_Success(t *testing.T) {
	devices := []Device{
		{SKU: "H5101", Device: "AA:BB:CC", DeviceName: "Sensor 1"},
		{SKU: "H5102", Device: "DD:EE:FF", DeviceName: "Sensor 2"},
	}
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/router/api/v1/user/devices" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get(apiKeyHeader) == "" {
			t.Error("missing API key header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(deviceListBody(t, devices))
	})

	got, err := client.ListDevices(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(devices): got %d want 2", len(got))
	}
	if got[0].SKU != "H5101" {
		t.Errorf("SKU: got %q want H5101", got[0].SKU)
	}
}

func TestListDevices_403(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":403,"message":"Invalid API key"}`, http.StatusForbidden)
	})
	_, err := client.ListDevices(context.Background(), "bad-key")
	if err == nil {
		t.Error("expected error for HTTP 403")
	}
}

func TestListDevices_429_FromServer(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message":"rate limited"}`))
	})
	_, err := client.ListDevices(context.Background(), "test-key")
	if err == nil {
		t.Error("expected error for HTTP 429")
	}
}

func TestListDevices_RateLimiterBlocks(t *testing.T) {
	// Exhaust internal rate limiter; server should never be hit.
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("request reached server after rate limit was exhausted")
	})
	client.rateLimiter.count = dailyRateLimit

	_, err := client.ListDevices(context.Background(), "test-key")
	if err == nil {
		t.Error("expected error when internal rate limit is exhausted")
	}
}

func TestListDevices_MalformedJSON(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not valid json`))
	})
	_, err := client.ListDevices(context.Background(), "test-key")
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestListDevices_APIErrorCode(t *testing.T) {
	body, _ := json.Marshal(DeviceListPayload{Code: 400, Message: "bad request"})
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})
	_, err := client.ListDevices(context.Background(), "test-key")
	if err == nil {
		t.Error("expected error for API error code 400")
	}
}

// ---------------------------------------------------------------------------
// QueryDeviceState
// ---------------------------------------------------------------------------

func TestQueryDeviceState_Success(t *testing.T) {
	caps := []StateCapabilityValue{
		{Instance: "temperature", State: 23.5},
		{Instance: "humidity", State: float64(55)},
	}
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(stateBody(t, caps))
	})

	state, err := client.QueryDeviceState(context.Background(), "test-key", "H5101", "AA:BB:CC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.Capabilities) != 2 {
		t.Fatalf("capabilities: got %d want 2", len(state.Capabilities))
	}
	if state.Capabilities[0].Instance != "temperature" {
		t.Errorf("instance: got %q want temperature", state.Capabilities[0].Instance)
	}
}

func TestQueryDeviceState_APIErrorCode(t *testing.T) {
	body, _ := json.Marshal(DeviceStatePayload{Code: 400, Message: "device not found"})
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})
	_, err := client.QueryDeviceState(context.Background(), "test-key", "H5101", "AA:BB:CC")
	if err == nil {
		t.Error("expected error for API error code 400")
	}
}

func TestQueryDeviceState_HTTPError(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})
	_, err := client.QueryDeviceState(context.Background(), "test-key", "H5101", "AA:BB:CC")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestQueryDeviceState_MalformedJSON(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not valid json`))
	})
	_, err := client.QueryDeviceState(context.Background(), "test-key", "H5101", "AA:BB:CC")
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

// TestQueryDeviceState_RealAPIShape verifies that the parser correctly handles
// the actual Govee state API response format, which uses "payload" and "msg"
// (not "data" and "message" like the device-list endpoint).
func TestQueryDeviceState_RealAPIShape(t *testing.T) {
	// Literal response mirroring the real Govee API for POST /device/state.
	realResponse := `{
		"requestId": "test-1",
		"msg": "success",
		"code": 200,
		"payload": {
			"sku": "H6609",
			"device": "1E:8C:DC:6E:02:86:24:83",
			"capabilities": [
				{"type": "devices.capabilities.online",    "instance": "online",      "state": {"value": true}},
				{"type": "devices.capabilities.on_off",    "instance": "powerSwitch", "state": {"value": 1}},
				{"type": "devices.capabilities.range",     "instance": "brightness",  "state": {"value": 100}}
			]
		}
	}`

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(realResponse))
	})

	state, err := client.QueryDeviceState(context.Background(), "test-key", "H6609", "1E:8C:DC:6E:02:86:24:83")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.Capabilities) != 3 {
		t.Fatalf("capabilities: got %d want 3", len(state.Capabilities))
	}
	if state.Capabilities[2].Instance != "brightness" {
		t.Errorf("instance: got %q want brightness", state.Capabilities[2].Instance)
	}
	// State is {"value": 100} — a nested object. toFloat64 should handle it.
	brightVal, err := toFloat64(state.Capabilities[2].State)
	if err != nil {
		t.Errorf("toFloat64(brightness state): %v", err)
	}
	if brightVal != 100 {
		t.Errorf("brightness value: got %v want 100", brightVal)
	}
}

// Package plugin contains the Govee API client and datasource backend logic.
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	defaultBaseURL      = "https://openapi.api.govee.com"
	dailyRateLimit      = 10_000
	apiKeyHeader        = "Govee-API-Key"
	contentTypeJSON     = "application/json"
	httpTimeoutSeconds  = 15
	defaultStateTTL     = 60 * time.Second
)

// ---------------------------------------------------------------------------
// Govee API data structures
// ---------------------------------------------------------------------------

// Capability represents a single device capability entry returned by the
// Govee /user/devices endpoint.
type Capability struct {
	Type       string                 `json:"type"`
	Instance   string                 `json:"instance"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// Device represents a Govee device as returned by the list-devices endpoint.
type Device struct {
	SKU          string       `json:"sku"`
	Device       string       `json:"device"`
	DeviceName   string       `json:"deviceName"`
	Type         string       `json:"type"`
	Capabilities []Capability `json:"capabilities"`
}

// DeviceListPayload is the outer response wrapper for GET /user/devices.
type DeviceListPayload struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []Device `json:"data"`
}

// ---------------------------------------------------------------------------
// Device state structures
// ---------------------------------------------------------------------------

// StateCapabilityValue holds a single capability's state as returned by the
// POST /device/state endpoint.
type StateCapabilityValue struct {
	Type     string      `json:"type"`
	Instance string      `json:"instance"`
	State    interface{} `json:"state"`
}

// DeviceStateData is the inner data object of the state response.
type DeviceStateData struct {
	SKU          string                 `json:"sku"`
	Device       string                 `json:"device"`
	Capabilities []StateCapabilityValue `json:"capabilities"`
}

// DeviceStatePayload is the outer response wrapper for POST /device/state.
type DeviceStatePayload struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    DeviceStateData `json:"data"`
}

// ---------------------------------------------------------------------------
// Rate limit tracker
// ---------------------------------------------------------------------------

// rateLimiter tracks daily API usage in-memory. The counter resets at midnight
// UTC. This is a best-effort guard — it resets when the plugin process restarts.
type rateLimiter struct {
	mu        sync.Mutex
	count     int
	resetDate string // YYYY-MM-DD in UTC
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{resetDate: todayUTC()}
}

func todayUTC() string {
	return time.Now().UTC().Format("2006-01-02")
}

// Allow returns true if the request is within the daily quota, and increments
// the counter. Returns false if the daily limit has been reached.
func (r *rateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	today := todayUTC()
	if today != r.resetDate {
		r.count = 0
		r.resetDate = today
	}

	if r.count >= dailyRateLimit {
		return false
	}
	r.count++
	return true
}

// Remaining returns the number of requests remaining today.
func (r *rateLimiter) Remaining() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	today := todayUTC()
	if today != r.resetDate {
		return dailyRateLimit
	}
	remaining := dailyRateLimit - r.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ---------------------------------------------------------------------------
// GoveeClient
// ---------------------------------------------------------------------------

// stateCacheEntry holds a cached device state response and its expiry time.
type stateCacheEntry struct {
	data      *DeviceStateData
	expiresAt time.Time
}

// GoveeClient wraps the Govee OpenAPI HTTP client.
// The API key is passed per-call (from DecryptedSecureJSONData) and is
// never stored in the struct so it cannot leak into logs.
type GoveeClient struct {
	baseURL     string
	httpClient  *http.Client
	rateLimiter *rateLimiter
	stateTTL    time.Duration
	cacheMu     sync.RWMutex
	stateCache  map[string]stateCacheEntry
}

// NewGoveeClient creates a new GoveeClient. baseURL may be empty, in which
// case the default Govee API URL is used.
func NewGoveeClient(baseURL string) *GoveeClient {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &GoveeClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: httpTimeoutSeconds * time.Second,
		},
		rateLimiter: newRateLimiter(),
		stateTTL:    defaultStateTTL,
		stateCache:  make(map[string]stateCacheEntry),
	}
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

// do executes an authenticated request against the Govee API. The API key is
// added only as an HTTP header and is never written to logs.
func (c *GoveeClient) do(ctx context.Context, method, path string, apiKey string, body interface{}) ([]byte, int, error) {
	if !c.rateLimiter.Allow() {
		return nil, http.StatusTooManyRequests, errors.New("govee: daily rate limit of 10,000 requests reached; resets at midnight UTC")
	}

	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("govee: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("govee: create request: %w", err)
	}

	req.Header.Set(apiKeyHeader, apiKey)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("govee: http request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("govee: read response body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, resp.StatusCode, errors.New("govee: API rate limit exceeded (HTTP 429)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("govee: API error HTTP %d: %s", resp.StatusCode, truncate(string(data), 256))
	}

	return data, resp.StatusCode, nil
}

// truncate cuts s to at most maxLen characters to avoid bloating error messages.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}

// ---------------------------------------------------------------------------
// ListDevices
// ---------------------------------------------------------------------------

// ListDevices calls GET /router/api/v1/user/devices and returns the slice of
// devices registered under the supplied API key.
func (c *GoveeClient) ListDevices(ctx context.Context, apiKey string) ([]Device, error) {
	data, _, err := c.do(ctx, http.MethodGet, "/router/api/v1/user/devices", apiKey, nil)
	if err != nil {
		return nil, err
	}

	var payload DeviceListPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("govee: parse device list: %w", err)
	}

	if payload.Code != 0 && payload.Code != 200 {
		return nil, fmt.Errorf("govee: API returned code %d: %s", payload.Code, payload.Message)
	}

	return payload.Data, nil
}

// ---------------------------------------------------------------------------
// QueryDeviceState
// ---------------------------------------------------------------------------

// stateRequest is the POST body for /device/state.
type stateRequest struct {
	RequestID string         `json:"requestId"`
	Payload   stateReqPayload `json:"payload"`
}

type stateReqPayload struct {
	SKU    string `json:"sku"`
	Device string `json:"device"`
}

// QueryDeviceState calls POST /router/api/v1/device/state and returns the
// device state for the given SKU and device identifier. Responses are cached
// for stateTTL (default 60s) to avoid burning the 10,000 req/day quota when
// multiple panels query the same device.
func (c *GoveeClient) QueryDeviceState(ctx context.Context, apiKey, sku, device string) (*DeviceStateData, error) {
	cacheKey := sku + ":" + device

	// Check cache under read lock.
	c.cacheMu.RLock()
	entry, ok := c.stateCache[cacheKey]
	c.cacheMu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.data, nil
	}

	reqBody := stateRequest{
		RequestID: fmt.Sprintf("%s-%s-%d", sku, device, time.Now().UnixNano()),
		Payload: stateReqPayload{
			SKU:    sku,
			Device: device,
		},
	}

	data, _, err := c.do(ctx, http.MethodPost, "/router/api/v1/device/state", apiKey, reqBody)
	if err != nil {
		return nil, err
	}

	var payload DeviceStatePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("govee: parse device state: %w", err)
	}

	if payload.Code != 0 && payload.Code != 200 {
		return nil, fmt.Errorf("govee: API returned code %d: %s", payload.Code, payload.Message)
	}

	// Store in cache under write lock.
	c.cacheMu.Lock()
	c.stateCache[cacheKey] = stateCacheEntry{
		data:      &payload.Data,
		expiresAt: time.Now().Add(c.stateTTL),
	}
	c.cacheMu.Unlock()

	return &payload.Data, nil
}

// ---------------------------------------------------------------------------
// RateLimitRemaining (for health check info)
// ---------------------------------------------------------------------------

// RateLimitRemaining returns how many API calls remain in the current UTC day.
func (c *GoveeClient) RateLimitRemaining() int {
	return c.rateLimiter.Remaining()
}
